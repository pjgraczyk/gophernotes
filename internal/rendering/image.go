package rendering

import (
	"bytes"
	"image"
	"image/png"
)

func Image(img image.Image) Data {
	b, mimeType, err := EncodePng(img)
	if err != nil {
		return MakeDataErr(err)
	}
	return Data{
		Data: MIMEMap{
			mimeType: b,
		},
		Metadata: MIMEMap{
			mimeType: ImageMetadata(img),
		},
	}
}

func EncodePng(img image.Image) (data []byte, mimeType string, err error) {
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), MIMETypePNG, nil
}

func ImageMetadata(img image.Image) MIMEMap {
	rect := img.Bounds()
	return MIMEMap{
		"width":  rect.Dx(),
		"height": rect.Dy(),
	}
}
