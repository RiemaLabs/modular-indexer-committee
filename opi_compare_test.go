package main

import (
	"testing"
)

func TestOPI(t *testing.T) {
	loadCatchUp(780000)
}


func goTOHeight() {
	getter, _ := loadMain()
	queue := loadCatchUp(780000)
	loadService(getter, queue, 5000)

}