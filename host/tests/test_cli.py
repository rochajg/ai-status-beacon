import sys
import pytest
from unittest.mock import patch
from beacon.cli import main


def run_cli(*args):
    with patch("sys.argv", ["beacon", *args]):
        with pytest.raises(SystemExit) as exc:
            main()
    return exc.value.code


def test_no_args_exits_0():
    assert run_cli() == 0


def test_help_exits_0():
    assert run_cli("--help") == 0


def test_invalid_state_exits_0():
    assert run_cli("flying") == 0


def test_direct_with_device_sends_state():
    with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
         patch("beacon.serialio.send_direct", return_value=True) as mock_send:
        code = run_cli("thinking", "--direct")
    mock_send.assert_called_once_with("thinking", "/dev/cu.fake", timeout_s=0.3)
    assert code == 0


def test_direct_exits_0_when_no_device():
    with patch("beacon.serialio.find_port", return_value=None):
        code = run_cli("done", "--direct")
    assert code == 0


def test_direct_exits_0_when_send_fails():
    with patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
         patch("beacon.serialio.send_direct", return_value=False):
        code = run_cli("idle", "--direct")
    assert code == 0


import pathlib
SOCKET_PATH = str(pathlib.Path.home() / ".beacon" / "beacon.sock")


def test_default_tries_socket_first():
    with patch("beacon.serialio.send_to_socket", return_value=True) as mock_sock, \
         patch("beacon.serialio.send_direct") as mock_direct:
        run_cli("thinking")
    mock_sock.assert_called_once_with("thinking", SOCKET_PATH, timeout_s=0.3)
    mock_direct.assert_not_called()


def test_default_silent_if_socket_fails_no_direct():
    with patch("beacon.serialio.send_to_socket", return_value=False), \
         patch("beacon.serialio.find_port", return_value=None):
        code = run_cli("thinking")
    assert code == 0


def test_direct_flag_skips_socket():
    with patch("beacon.serialio.send_to_socket") as mock_sock, \
         patch("beacon.serialio.find_port", return_value="/dev/cu.fake"), \
         patch("beacon.serialio.send_direct", return_value=True) as mock_direct:
        run_cli("done", "--direct")
    mock_sock.assert_not_called()
    mock_direct.assert_called_once_with("done", "/dev/cu.fake", timeout_s=0.3)
