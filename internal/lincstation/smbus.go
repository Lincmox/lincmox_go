package lincstation

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// I2C_SLAVE is the ioctl command to set the I2C slave address.
const I2C_SLAVE = 0x0703

const (
	I2C_SMBUS           = 0x0720
	I2C_SMBUS_QUICK     = 0
	I2C_SMBUS_WRITE     = 0
	I2C_SMBUS_READ      = 1
	I2C_SMBUS_BYTE_DATA = 2
)

type i2cSmbusIoctlData struct {
	readWrite byte
	command   byte
	size      uint32
	data      uintptr
}

// smbusDevice implements i2cBus using Linux I2C ioctl.
type smbusDevice struct {
	fd      *os.File
	busNum  int
	address byte
	verbose bool
}

// newSMBusDevice opens the I2C bus and sets the slave address.
// If busID is -1 (default), it auto-detects the bus by scanning for the device at address 0x26.
func newSMBusDevice(busID int, verbose bool) (*smbusDevice, error) {
	var busNum int
	var err error

	if busID >= 0 {
		busNum = busID
	} else {
		busNum, err = detectBus(verbose)
		if err != nil {
			return nil, err
		}
	}

	path := fmt.Sprintf("/dev/i2c-%d", busNum)
	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	// Set slave address
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), I2C_SLAVE, uintptr(i2cAddress)); errno != 0 {
		fd.Close()
		return nil, fmt.Errorf("set slave address 0x%02X on bus %d: %v", i2cAddress, busNum, errno)
	}

	if verbose {
		fmt.Printf("[I2C] Opened bus %d at %s, slave 0x%02X\n", busNum, path, i2cAddress)
	}

	return &smbusDevice{
		fd:      fd,
		busNum:  busNum,
		address: i2cAddress,
		verbose: verbose,
	}, nil
}

// detectBus scans I2C buses 0-9 to find the device at address 0x26.
func detectBus(verbose bool) (int, error) {
	for bus := 0; bus <= 9; bus++ {
		path := fmt.Sprintf("/dev/i2c-%d", bus)
		fd, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			continue
		}

		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), I2C_SLAVE, uintptr(i2cAddress))
		if errno != 0 {
			fd.Close()
			continue
		}

		// First try a real I2C_RDWR write. This forces the host controller to
		// issue a START + address + write on the bus and wait for the slave's ACK.
		// If no device responds at 0x26 the kernel returns EREMOTEIO/EIO, so a
		// bus with a stray device at another address (or a controller that ACKs
		// reads on an idle bus) will correctly be rejected.
		// A plain fd.Read() is unreliable here: several Linux I2C drivers succeed
		// on a read even when no slave ACKs, producing a false positive on the
		// first bus scanned.
		buf := []byte{0x00}
		msgs := []i2cMsg{{
			addr:  uint16(i2cAddress),
			flags: 0,
			len:   uint16(len(buf)),
			buf:   uintptr(unsafe.Pointer(&buf[0])),
		}}
		data := i2cRdwrIoctlData{
			msgs:  &msgs[0],
			nmsgs: 1,
		}

		found := false
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), I2C_RDWR, uintptr(unsafe.Pointer(&data))); errno == 0 {
			found = true
		} else {
			// Fall back to a SMBus "send byte" (quick write) probe, the same
			// mechanism i2cdetect uses by default. Some slave devices ACK only a
			// bare START + address + W with no data byte, and would otherwise be
			// missed by the I2C_RDWR write above. This path still performs a real
			// bus transaction so it keeps the false-positive protection.
			smbus := i2cSmbusIoctlData{
				readWrite: I2C_SMBUS_WRITE,
				command:   0,
				size:      I2C_SMBUS_QUICK,
				data:      0,
			}
			if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&smbus))); errno == 0 {
				found = true
			} else {
				// Finally, probe with a real SMBus register write (command byte +
				// data byte), the exact protocol the LincStation controller
				// answers. A bare write NACKs, and some adapters (Synopsys
				// DesignWare) reject the zero-length quick write above via the
				// I2C_AQ_NO_ZERO_LEN quirk, so neither earlier probe can see a
				// present device on those buses. Writing 0x00 to the LED-off
				// register (0xB0) is harmless. SMBus-less adapters (e.g. i915
				// gmbus) return EOPNOTSUPP here and are skipped, so no false
				// positive is added on their buses.
				probeBuf := []byte{0x00}
				registerWrite := i2cSmbusIoctlData{
					readWrite: I2C_SMBUS_WRITE,
					command:   0xB0,
					size:      I2C_SMBUS_BYTE_DATA,
					data:      uintptr(unsafe.Pointer(&probeBuf[0])),
				}
				if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&registerWrite))); errno == 0 {
					found = true
				}
			}
		}
		fd.Close()

		if found {
			if verbose {
				fmt.Printf("[I2C] Auto-detected device at bus %d\n", bus)
			}
			return bus, nil
		}
	}
	return 0, ErrBusNotFound
}

// ReadByte reads a single byte from the given register using SMBUS ioctl.
func (d *smbusDevice) ReadByte(reg byte) (byte, error) {
	buf := make([]byte, 34)
	args := i2cSmbusIoctlData{
		readWrite: I2C_SMBUS_READ,
		command:   reg,
		size:      I2C_SMBUS_BYTE_DATA,
		data:      uintptr(unsafe.Pointer(&buf[0])),
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&args)))
	if errno != 0 {
		return 0, errno
	}
	return buf[0], nil
}

// WriteByte writes a single byte to the given register using SMBUS ioctl.
func (d *smbusDevice) WriteByte(reg, value byte) error {
	buf := make([]byte, 34)
	buf[0] = value
	args := i2cSmbusIoctlData{
		readWrite: I2C_SMBUS_WRITE,
		command:   reg,
		size:      I2C_SMBUS_BYTE_DATA,
		data:      uintptr(unsafe.Pointer(&buf[0])),
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&args)))
	if errno != 0 {
		return errno
	}
	return nil
}

// Close closes the I2C device file.
func (d *smbusDevice) Close() error {
	if d.fd != nil {
		if d.verbose {
			fmt.Printf("[I2C] Closed bus %d\n", d.busNum)
		}
		return d.fd.Close()
	}
	return nil
}

// ioctl is a helper for I2C ioctl calls (unused in current implementation but kept for reference).
func ioctl(fd uintptr, request uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// i2cMsg represents an I2C message for ioctl I2C_RDWR.
type i2cMsg struct {
	addr  uint16
	flags uint16
	len   uint16
	buf   uintptr
}

// i2cRdwrIoctlData is the data structure for I2C_RDWR ioctl.
type i2cRdwrIoctlData struct {
	msgs  *i2cMsg
	nmsgs uint32
}

// I2C_RDWR is the ioctl command for combined read/write.
const I2C_RDWR = 0x0707

// I2C_M_RD is the flag for read direction.
const I2C_M_RD = 0x0001

// ReadWrite performs a combined I2C read/write transaction using I2C_RDWR ioctl.
// This is more reliable than separate Write/Read for register-based devices.
func (d *smbusDevice) ReadWrite(writeBuf, readBuf []byte) error {
	if len(writeBuf) == 0 && len(readBuf) == 0 {
		return nil
	}

	msgs := make([]i2cMsg, 0, 2)
	if len(writeBuf) > 0 {
		msgs = append(msgs, i2cMsg{
			addr:  uint16(d.address),
			flags: 0,
			len:   uint16(len(writeBuf)),
			buf:   uintptr(unsafe.Pointer(&writeBuf[0])),
		})
	}
	if len(readBuf) > 0 {
		msgs = append(msgs, i2cMsg{
			addr:  uint16(d.address),
			flags: I2C_M_RD,
			len:   uint16(len(readBuf)),
			buf:   uintptr(unsafe.Pointer(&readBuf[0])),
		})
	}

	data := i2cRdwrIoctlData{
		msgs:  &msgs[0],
		nmsgs: uint32(len(msgs)),
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), I2C_RDWR, uintptr(unsafe.Pointer(&data)))
	if errno != 0 {
		return errno
	}
	return nil
}