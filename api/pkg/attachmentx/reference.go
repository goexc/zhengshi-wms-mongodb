// Package attachmentx preserves the existing images/ relative-reference contract.
package attachmentx

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

const MaxFiles = 10
const MaxFileBytes int64 = 20 << 20
const MaxRequestBytes int64 = MaxFileBytes + (1 << 20)

func Prefix(scope string) (string, error) {
	switch scope {
	case "":
		return "", nil
	case "acceptance":
		return "acceptance/", nil
	case "acceptance_redemption":
		return "acceptance/redemption/", nil
	case "invoice":
		return "invoice/", nil
	default:
		return "", fmt.Errorf("不支持的附件业务分类")
	}
}

func Scope(reference string) string {
	switch {
	case strings.HasPrefix(reference, "acceptance/redemption/"):
		return "acceptance_redemption"
	case strings.HasPrefix(reference, "acceptance/"):
		return "acceptance"
	case strings.HasPrefix(reference, "invoice/"):
		return "invoice"
	default:
		return ""
	}
}

func ValidateReference(reference string) error {
	if reference == "" || strings.TrimSpace(reference) != reference || len(reference) > 512 || strings.HasPrefix(reference, "/") || strings.HasPrefix(reference, "images/") || strings.ContainsAny(reference, "\\,:?#%\x00\r\n") {
		return fmt.Errorf("附件引用格式错误")
	}
	for _, part := range strings.Split(reference, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("附件引用格式错误")
		}
	}
	return nil
}

func ValidateReferences(references []string) ([]string, error) {
	if len(references) > MaxFiles {
		return nil, fmt.Errorf("每组附件最多%d个", MaxFiles)
	}
	result := make([]string, 0, len(references))
	seen := make(map[string]bool)
	for _, reference := range references {
		if err := ValidateReference(reference); err != nil {
			return nil, err
		}
		if !seen[reference] {
			result = append(result, reference)
			seen[reference] = true
		}
	}
	return result, nil
}

// ValidateScopedReferences allows the legacy flat/unspecified directory contract
// while preventing new financial files from being attached under the wrong use.
// Existing references may be retained without moving or rewriting old records.
func ValidateScopedReferences(references []string, expectedScope string, existing []string) error {
	if _, err := Prefix(expectedScope); err != nil || expectedScope == "" {
		return fmt.Errorf("附件业务分类不合法")
	}
	clean, err := ValidateReferences(references)
	if err != nil {
		return err
	}
	previous := make(map[string]bool, len(existing))
	for _, reference := range existing {
		previous[reference] = true
	}
	for _, reference := range clean {
		if actual := Scope(reference); actual != "" && actual != expectedScope && !previous[reference] {
			return fmt.Errorf("附件用途与当前单据不一致，请在对应附件区域上传")
		}
	}
	return nil
}

func NewReference(scope, filename string, now time.Time) (string, error) {
	prefix, err := Prefix(scope)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(filename, "\\", "/")))
	if scope != "" {
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".pdf", ".ofd", ".doc", ".docx", ".xls", ".xlsx":
		default:
			return "", fmt.Errorf("支持图片、PDF、OFD、Word和Excel文件")
		}
		prefix += now.Format("2006/01/")
	} else if strings.ContainsAny(ext, ",:?#%\x00\r\n") {
		return "", fmt.Errorf("文件扩展名不合法")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(id[:]) + ext, nil
}
