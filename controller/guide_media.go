package controller

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	guideImageMaxBytes  = 10 << 20
	guideVideoMaxBytes  = 100 << 20
	guideUploadBodySize = guideVideoMaxBytes + (1 << 20)
)

func guideMediaDirectory() string {
	return common.GetEnvOrDefaultString("GUIDE_MEDIA_DIR", "data/guides-media")
}

var guideMediaTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".ogv":  "video/ogg",
}

func guideMediaType(filename, declared string) (string, int64, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	contentType, ok := guideMediaTypes[ext]
	if !ok {
		return "", 0, false
	}
	declared = strings.ToLower(strings.TrimSpace(strings.Split(declared, ";")[0]))
	if declared == "" {
		return "", 0, false
	}
	if parsed, _, err := mime.ParseMediaType(declared); err == nil {
		declared = parsed
	}
	if declared != contentType {
		return "", 0, false
	}
	if strings.HasPrefix(contentType, "image/") {
		return contentType, guideImageMaxBytes, true
	}
	return contentType, guideVideoMaxBytes, true
}

// UploadGuideMedia stores an administrator-uploaded image or video for use in guides.
func UploadGuideMedia(c *gin.Context) {
	// Cap the whole multipart request before parsing it. The per-type limit is
	// checked against the multipart file header below.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, guideUploadBodySize)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择要上传的图片或视频"})
		return
	}
	contentType, maxBytes, ok := guideMediaType(fileHeader.Filename, fileHeader.Header.Get("Content-Type"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "仅支持 JPG、PNG、GIF、WEBP、MP4、WEBM、MOV 或 OGV 文件"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": fmt.Sprintf("文件大小不能超过 %d MB", maxBytes/(1<<20))})
		return
	}

	mediaDir := guideMediaDirectory()
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		common.ApiError(c, err)
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	filename := uuid.NewString() + ext
	path := filepath.Join(mediaDir, filename)
	src, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	_, copyErr := io.Copy(dst, io.LimitReader(src, maxBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if copyErr != nil {
			common.ApiError(c, copyErr)
		} else {
			common.ApiError(c, closeErr)
		}
		return
	}
	if info, statErr := os.Stat(path); statErr != nil || info.Size() > maxBytes {
		_ = os.Remove(path)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "文件大小超过限制"})
		return
	}

	// This endpoint is consumed by the guide editor, which expects the URL at
	// the top level rather than inside the generic `data` envelope.
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "",
		"url":          "/api/guides/media/" + filename,
		"filename":     filename,
		"content_type": contentType,
	})
}

// GetGuideMedia serves a previously uploaded guide image or video.
func GetGuideMedia(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) {
		c.Status(http.StatusNotFound)
		return
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := guideMediaTypes[ext]; !ok {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(guideMediaDirectory(), filename)
	if _, err := os.Stat(path); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(path)
}
