package handler

import (
	"github.com/suixibing/cocom/pkg/imaging/webp"
)

func init() {
	mux.HandleFunc(webp.InstallScriptEndpoint, webp.HandleWebPInstall)
}
