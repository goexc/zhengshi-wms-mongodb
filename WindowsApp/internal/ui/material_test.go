package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/lxn/win"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestMaterialDrawingStatus(t *testing.T) {
	if got := materialDrawingStatus(api.Material{Image: "drawing.png"}); got != "有图纸" {
		t.Fatalf("drawing status = %q", got)
	}
	if got := materialDrawingStatus(api.Material{Image: "  "}); got != "无图纸" {
		t.Fatalf("empty drawing status = %q", got)
	}
}

func TestMaterialInfoTextIncludesBusinessFields(t *testing.T) {
	text := materialInfoText(api.Material{
		Name: "测试物料", Model: "M-01", CategoryName: "标准件",
		Material: "碳钢", Specification: "100×20", SurfaceTreatment: "镀锌",
		StrengthGrade: "8.8", Quantity: 12, Unit: "件", Remark: "测试备注",
	})
	for _, want := range []string{"测试物料", "M-01", "标准件", "碳钢", "100×20", "镀锌", "8.8", "12 件", "测试备注"} {
		if !strings.Contains(text, want) {
			t.Fatalf("material info does not contain %q: %q", want, text)
		}
	}
}

func TestMaterialZoomCalculations(t *testing.T) {
	bounds := image.Rect(0, 0, 1200, 800)
	fit := materialFitWidthZoom(bounds, 600)
	if math.Abs(fit-0.5) > 0.001 {
		t.Fatalf("fit-width zoom = %f", fit)
	}
	tallDrawingFit := materialFitWidthZoom(image.Rect(0, 0, 1200, 4800), 600)
	if math.Abs(tallDrawingFit-0.5) > 0.001 {
		t.Fatalf("fit-width zoom must not depend on image height: %f", tallDrawingFit)
	}
	if got := nextMaterialZoom(fit, 1, 4); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("next zoom in = %f", got)
	}
	if got := nextMaterialZoom(fit, -1, 4); math.Abs(got-0.4) > 0.001 {
		t.Fatalf("next zoom out = %f", got)
	}
}

func TestMaterialDialogWindowStyle(t *testing.T) {
	baseStyle := uint32(win.WS_CAPTION | win.WS_SYSMENU | win.WS_THICKFRAME)
	style := materialDialogWindowStyle(baseStyle)
	if style&win.WS_MINIMIZEBOX == 0 {
		t.Fatal("material dialog style does not include minimize box")
	}
	if style&win.WS_MAXIMIZEBOX == 0 {
		t.Fatal("material dialog style does not include maximize box")
	}
	if style&baseStyle != baseStyle {
		t.Fatalf("material dialog style lost existing bits: %#x", style)
	}
}

func TestScaleMaterialDrawing(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	scaled := scaleMaterialDrawing(source, 1.5)
	if scaled.Bounds().Dx() != 150 || scaled.Bounds().Dy() != 120 {
		t.Fatalf("scaled bounds = %v", scaled.Bounds())
	}
}

func TestRotateMaterialDrawing(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	red := color.NRGBA{R: 255, A: 255}
	green := color.NRGBA{G: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
	yellow := color.NRGBA{R: 255, G: 255, A: 255}
	source.SetNRGBA(0, 0, red)
	source.SetNRGBA(1, 0, green)
	source.SetNRGBA(0, 2, blue)
	source.SetNRGBA(1, 2, yellow)

	right := rotateMaterialDrawing(source, 1)
	if right.Bounds().Dx() != 3 || right.Bounds().Dy() != 2 {
		t.Fatalf("right-rotated bounds = %v", right.Bounds())
	}
	assertMaterialPixel(t, right, 2, 0, red)
	assertMaterialPixel(t, right, 2, 1, green)
	assertMaterialPixel(t, right, 0, 0, blue)
	assertMaterialPixel(t, right, 0, 1, yellow)

	left := rotateMaterialDrawing(source, -1)
	if left.Bounds().Dx() != 3 || left.Bounds().Dy() != 2 {
		t.Fatalf("left-rotated bounds = %v", left.Bounds())
	}
	assertMaterialPixel(t, left, 0, 1, red)
	assertMaterialPixel(t, left, 0, 0, green)
	assertMaterialPixel(t, left, 2, 1, blue)
	assertMaterialPixel(t, left, 2, 0, yellow)
}

func TestMaterialRotationStatus(t *testing.T) {
	cases := []struct {
		turns int
		want  string
	}{
		{0, "图纸已加载"},
		{1, "图纸已加载 · 已向右旋转 90°"},
		{2, "图纸已加载 · 已旋转 180°"},
		{-1, "图纸已加载 · 已向左旋转 90°"},
		{4, "图纸已加载"},
	}
	for _, tc := range cases {
		if got := materialDrawingReadyStatus(tc.turns); got != tc.want {
			t.Fatalf("rotation status for %d = %q, want %q", tc.turns, got, tc.want)
		}
	}
}

func assertMaterialPixel(t *testing.T, drawing image.Image, x, y int, want color.NRGBA) {
	t.Helper()
	got := color.NRGBAModel.Convert(drawing.At(x, y)).(color.NRGBA)
	if got != want {
		t.Fatalf("pixel (%d,%d) = %#v, want %#v", x, y, got, want)
	}
}
