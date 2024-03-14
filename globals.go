package main

// The number of confirmations to be considered immutable and can't be re-organized.
// Keep the same with OPI.
// TODO: Find the original code of OPI.
const BitcoinConfirmations uint = 10

// Version Control
var Version string

var GlobalConfig Config
