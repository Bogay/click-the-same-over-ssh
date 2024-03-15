package main

type BlockFlags struct {
	user       string
	row        int
	col        int
	isHovered  bool
	isSelected bool
}

func updateBlockFlags(row, col int, block *ArithmeticBlock) BlockFlags {
	return BlockFlags{
		row:        row,
		col:        col,
		isHovered:  block.isHovered,
		isSelected: block.isSelected,
	}
}

type Join struct {
	user  string
	index int
}

type Score struct {
	user  string
	delta int
}
