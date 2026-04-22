package mexpr

import (
	"reflect"
	"testing"
)

func TestToNumber(t *testing.T) {
	ast := &Node{}
	cases := []struct {
		name  string
		value any
		want  float64
	}{
		{name: "int32", value: int32(3), want: 3},
		{name: "uint64", value: uint64(4), want: 4},
		{name: "float32", value: float32(1.5), want: 1.5},
		{name: "func int16", value: func() int16 { return 7 }, want: 7},
		{name: "func uint32", value: func() uint32 { return 8 }, want: 8},
		{name: "func float32", value: func() float32 { return 2.5 }, want: 2.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := toNumber(ast, tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("expected %v but found %v", tc.want, got)
			}
		})
	}
}

func TestAppendAndConcatGenericSlices(t *testing.T) {
	appended, ok := appendSliceItems(nil, [3]uint{1, 2, 3})
	if !ok {
		t.Fatal("expected array append to succeed")
	}
	if !reflect.DeepEqual([]any{uint(1), uint(2), uint(3)}, appended) {
		t.Fatalf("unexpected appended items: %v", appended)
	}

	concatenated, ok := concatSlices([2]uint{1, 2}, [2]uint{3, 4})
	if !ok {
		t.Fatal("expected array concat to succeed")
	}
	if !reflect.DeepEqual([]any{uint(1), uint(2), uint(3), uint(4)}, concatenated) {
		t.Fatalf("unexpected concatenated items: %v", concatenated)
	}
}

func TestToBool(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "int16 positive", value: int16(1), want: true},
		{name: "uint32 zero", value: uint32(0), want: false},
		{name: "float32 positive", value: float32(1.25), want: true},
		{name: "bytes empty", value: []byte{}, want: false},
		{name: "bytes non-empty", value: []byte("x"), want: true},
		{name: "map any empty", value: map[any]any{}, want: false},
		{name: "map any non-empty", value: map[any]any{"k": 1}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toBool(tc.value)
			if got != tc.want {
				t.Fatalf("expected %v but found %v", tc.want, got)
			}
		})
	}
}