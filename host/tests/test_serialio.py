from unittest.mock import patch, MagicMock
import beacon.serialio as sio


def test_find_port_env_override(monkeypatch):
    monkeypatch.setenv("BEACON_SERIAL_PORT", "/dev/fake")
    assert sio.find_port() == "/dev/fake"


def test_find_port_glob(monkeypatch):
    monkeypatch.delenv("BEACON_SERIAL_PORT", raising=False)
    with patch("glob.glob", return_value=["/dev/cu.usbmodem42"]):
        assert sio.find_port() == "/dev/cu.usbmodem42"


def test_find_port_none_when_no_device(monkeypatch):
    monkeypatch.delenv("BEACON_SERIAL_PORT", raising=False)
    with patch("glob.glob", return_value=[]):
        assert sio.find_port() is None


def test_send_direct_writes_state():
    mock_serial = MagicMock()
    with patch("serial.Serial", return_value=mock_serial):
        result = sio.send_direct("thinking", "/dev/cu.fake")
    mock_serial.write.assert_called_once_with(b"thinking\n")
    mock_serial.close.assert_called_once()
    assert result is True


def test_send_direct_returns_false_on_error():
    with patch("serial.Serial", side_effect=OSError("no device")):
        result = sio.send_direct("thinking", "/dev/cu.fake")
    assert result is False


def test_send_direct_closes_port_on_exception():
    mock_serial = MagicMock()
    mock_serial.write.side_effect = OSError("write failed")
    with patch("serial.Serial", return_value=mock_serial):
        result = sio.send_direct("thinking", "/dev/cu.fake")
    mock_serial.close.assert_called_once()
    assert result is False
