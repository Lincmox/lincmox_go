package lincstation

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// I2C_SLAVE is the ioctl command to set the I2C slave address.
const I2C_SLAVE = 0x0703

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
		fd.Close()

		if errno == 0 {
			if verbose {
				fmt.Printf("[I2C] Auto-detected device at bus %d\n", bus)
			}
			return bus, nil
		}
	}
	return 0, ErrBusNotFound
}

// ReadByte reads a single byte from the given register.
func (d *smbusDevice) ReadByte(reg byte) (byte, error) {
	var val byte
	buf := []byte{reg}
	_, err := d.fd.Write(buf)
	if err != nil {
		return 0, err
	}
	_, err = d.fd.Read([]byte{val})
	if err != nil {
		return 0, err
	}
	return val, nil
}

// WriteByte writes a single byte to the given register.
func (d *smbusDevice) WriteByte(reg, value byte) error {
	buf := []byte{reg, value}
	_, err := d.fd.Write(buf)
	return err
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