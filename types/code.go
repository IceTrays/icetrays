package types

type InstructionCode = uint32

const (
	InstructionCP InstructionCode = iota
	InstructionMV
	InstructionRM
	InstructionMKDIR
	InstructionPIN
	InstructionUNPIN
)
