// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package color

import (
	"testing"
)

func delta(x, y uint8) uint8 {
	if x >= y {
		return x - y
	}
	return y - x
}

// Test that a subset of RGB space can be converted to Hex and back to within
// 1/256 tolerance.
func TestHex(t *testing.T) {
	for r := 0; r < 255; r += 7 {
		for g := 0; g < 255; g += 5 {
			for b := 0; b < 255; b += 3 {
				r0, g0, b0 := uint8(r), uint8(g), uint8(b)
				hex := RGBToHex(r0, g0, b0)
				r1, g1, b1 := HexToRGB(hex)
				if delta(r0, r1) > 1 || delta(g0, g1) > 1 || delta(b0, b1) > 1 {
					t.Fatalf("r0, g0, b0 = %d, %d, %d   r1, g1, b1 = %d, %d, %d", r0, g0, b0, r1, g1, b1)
				}
			}
		}
	}
}

// Test that a subset of RGB space can be converted to HSL and back to within
// 1/256 tolerance.
func TestHSL(t *testing.T) {
	for r := 0; r < 255; r += 7 {
		for g := 0; g < 255; g += 5 {
			for b := 0; b < 255; b += 3 {
				r0, g0, b0 := uint8(r), uint8(g), uint8(b)
				h, s, l := RGBToHSL(r0, g0, b0)
				r1, g1, b1 := HSLToRGB(h, s, l)
				if delta(r0, r1) > 1 || delta(g0, g1) > 1 || delta(b0, b1) > 1 {
					t.Fatalf("r0, g0, b0 = %d, %d, %d   r1, g1, b1 = %d, %d, %d", r0, g0, b0, r1, g1, b1)
				}
			}
		}
	}
}

// Test that a subset of RGB space can be converted to HSV and back to within
// 1/256 tolerance.
func TestHSV(t *testing.T) {
	for r := 0; r < 255; r += 7 {
		for g := 0; g < 255; g += 5 {
			for b := 0; b < 255; b += 3 {
				r0, g0, b0 := uint8(r), uint8(g), uint8(b)
				h, s, v := RGBToHSV(r0, g0, b0)
				r1, g1, b1 := HSVToRGB(h, s, v)
				if delta(r0, r1) > 1 || delta(g0, g1) > 1 || delta(b0, b1) > 1 {
					t.Fatalf("r0, g0, b0 = %d, %d, %d   r1, g1, b1 = %d, %d, %d", r0, g0, b0, r1, g1, b1)
				}
			}
		}
	}
}
