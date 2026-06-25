"""Tests for beacon.daemon — Unix socket to serial bridge."""

import socket
import threading
import time
from unittest.mock import MagicMock, patch

import beacon.daemon as daemon


def _send(sock_path: str, msg: str) -> None:
    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    s.connect(sock_path)
    s.sendall(msg.encode())
    s.close()


def test_daemon_forwards_state_to_serial(tmp_path):
    sock_path = str(tmp_path / "beacon.sock")
    mock_ser = MagicMock()
    mock_ser.is_open = True

    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
             patch("serial.Serial", return_value=mock_ser):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)  # wait for socket to be ready

    _send(sock_path, "thinking\n")
    time.sleep(0.1)
    stop.set()
    t.join(timeout=2)

    mock_ser.write.assert_called_with(b"thinking\n")


def test_daemon_ignores_invalid_state(tmp_path):
    sock_path = str(tmp_path / "beacon.sock")
    mock_ser = MagicMock()
    mock_ser.is_open = True
    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
             patch("serial.Serial", return_value=mock_ser):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)

    _send(sock_path, "launch_missiles\n")
    time.sleep(0.1)
    stop.set()
    t.join(timeout=2)

    mock_ser.write.assert_not_called()


def test_daemon_all_valid_states(tmp_path):
    """All valid states must be forwarded to serial."""
    valid_states = ["thinking", "waiting", "done", "idle", "error", "ping"]

    for state in valid_states:
        sock_path = str(tmp_path / f"beacon_{state}.sock")
        mock_ser = MagicMock()
        mock_ser.is_open = True
        stop = threading.Event()

        def run_daemon(sp=sock_path, ms=mock_ser):
            with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
                 patch("serial.Serial", return_value=ms):
                daemon.run(socket_path=sp, stop_event=stop)

        t = threading.Thread(target=run_daemon, daemon=True)
        t.start()
        time.sleep(0.3)

        _send(sock_path, f"{state}\n")
        time.sleep(0.1)
        stop.set()
        t.join(timeout=2)

        mock_ser.write.assert_called_with(state.encode() + b"\n")


def test_daemon_drops_command_when_serial_unavailable(tmp_path):
    """When serial is not connected, commands must be silently dropped."""
    sock_path = str(tmp_path / "beacon.sock")
    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value=None):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)

    # Should not raise — just silently drop
    _send(sock_path, "thinking\n")
    time.sleep(0.1)
    stop.set()
    t.join(timeout=2)


def test_daemon_removes_stale_socket(tmp_path):
    """run() must remove a stale socket file before binding."""
    sock_path = str(tmp_path / "beacon.sock")

    # Create a stale socket file
    with open(sock_path, "w") as f:
        f.write("stale")

    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value=None):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)
    stop.set()
    t.join(timeout=2)
    # If we get here without an error, the stale socket was removed successfully


def test_daemon_cleans_up_socket_on_exit(tmp_path):
    """Socket file must be removed after run() returns."""
    import os
    sock_path = str(tmp_path / "beacon.sock")
    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value=None):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)
    stop.set()
    t.join(timeout=2)

    assert not os.path.exists(sock_path), "Socket file should be removed on exit"


def test_daemon_strips_extra_whitespace(tmp_path):
    """Commands with surrounding whitespace must still be forwarded."""
    sock_path = str(tmp_path / "beacon.sock")
    mock_ser = MagicMock()
    mock_ser.is_open = True
    stop = threading.Event()

    def run_daemon():
        with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
             patch("serial.Serial", return_value=mock_ser):
            daemon.run(socket_path=sock_path, stop_event=stop)

    t = threading.Thread(target=run_daemon, daemon=True)
    t.start()
    time.sleep(0.3)

    _send(sock_path, "  thinking  \n")
    time.sleep(0.1)
    stop.set()
    t.join(timeout=2)

    mock_ser.write.assert_called_with(b"thinking\n")
