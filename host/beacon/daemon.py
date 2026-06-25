"""Daemon: holds the serial port open and bridges a Unix socket to it."""

import logging
import os
import pathlib
import socket
import threading
import time

import serial

from beacon import serialio

log = logging.getLogger(__name__)

VALID_STATES = {b"thinking", b"waiting", b"done", b"idle", b"error", b"ping"}
SOCKET_PATH = str(pathlib.Path.home() / ".beacon" / "beacon.sock")
RECONNECT_INTERVAL = 2.0


def run(
    socket_path: str = SOCKET_PATH,
    baud: int = 115200,
    stop_event: threading.Event | None = None,
) -> None:
    """Main daemon loop. Blocks until stop_event is set (or forever)."""
    pathlib.Path(socket_path).parent.mkdir(parents=True, exist_ok=True)

    # Remove stale socket file if present
    try:
        os.unlink(socket_path)
    except FileNotFoundError:
        pass

    srv = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    srv.bind(socket_path)
    srv.listen(8)
    srv.settimeout(0.5)  # allows the loop to periodically check stop_event

    ser: serial.Serial | None = None
    log.info("Daemon listening on %s", socket_path)

    try:
        while stop_event is None or not stop_event.is_set():
            # Keep serial connection alive
            if ser is None or not ser.is_open:
                ser = _connect_serial(baud)

            try:
                conn, _ = srv.accept()
            except socket.timeout:
                continue

            threading.Thread(
                target=_handle_connection,
                args=(conn, ser),
                daemon=True,
            ).start()
    finally:
        srv.close()
        try:
            os.unlink(socket_path)
        except FileNotFoundError:
            pass
        if ser is not None and ser.is_open:
            ser.close()


def _connect_serial(baud: int) -> "serial.Serial | None":
    """Discover the beacon serial port and open it. Returns None on failure."""
    port = serialio.find_port()
    if port is None:
        return None
    try:
        ser = serial.Serial(port, baud, timeout=0.1)
        log.info("Serial connected: %s", port)
        return ser
    except Exception as exc:
        log.warning(
            "Serial open failed (%s), retrying in %.1fs", exc, RECONNECT_INTERVAL
        )
        time.sleep(RECONNECT_INTERVAL)
        return None


def _handle_connection(conn: socket.socket, ser: "serial.Serial | None") -> None:
    """Read one command from a client connection and forward it to serial."""
    try:
        data = conn.recv(64).strip()
        if not data:
            return
        cmd = data.split()[0]
        if cmd not in VALID_STATES:
            log.debug("Ignored unknown command: %r", cmd)
            return
        if ser is None or not ser.is_open:
            log.warning("Serial not available, dropping command %r", cmd)
            return
        ser.write(cmd + b"\n")
    except Exception as exc:
        log.warning("Connection handler error: %s", exc)
    finally:
        conn.close()
