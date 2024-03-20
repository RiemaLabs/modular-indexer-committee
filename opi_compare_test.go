package main

import (
	"testing"
)

func TestOPI(t *testing.T) {
	var catchupHeight uint = 780000
	var toHeight uint = 785000
	goTOHeight(toHeight, catchupHeight)
}


func goTOHeight(toHeight uint, startHeight uint) {
	getter, _ := loadMain()
	queue := loadCatchUp(startHeight)
	loadService(getter, queue, toHeight-startHeight)
}