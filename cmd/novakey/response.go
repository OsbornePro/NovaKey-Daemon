package main


const respVersion byte = 3

type RespStatus byte

const (
	StatusOK           RespStatus = 0x00
	StatusNotArmed     RespStatus = 0x01
	StatusNeedsApprove RespStatus = 0x02
	StatusNotPaired    RespStatus = 0x03
	StatusBadRequest   RespStatus = 0x04
	StatusBadTimestamp RespStatus = 0x05
	StatusReplay       RespStatus = 0x06
	StatusRateLimit    RespStatus = 0x07
	StatusCryptoFail   RespStatus = 0x08

	// New: injection couldn't be performed, but we successfully copied to clipboard.
	StatusOKClipboard RespStatus = 0x09

	StatusInternal RespStatus = 0x7F
)

