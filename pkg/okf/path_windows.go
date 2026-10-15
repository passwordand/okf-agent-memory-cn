package okf

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Windows 的目录 junction 可能标记为 ModeIrregular，EvalSymlinks 不会展开。
// 通过文件句柄取得最终路径，统一解析符号链接、junction 和祖先重解析点。
func resolveExistingPath(name string) (string, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	encoded, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(encoded, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, 512)
	for {
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
		if err != nil {
			return "", err
		}
		if n >= uint32(len(buffer)) {
			buffer = make([]uint16, n+1)
			continue
		}
		resolved := windows.UTF16ToString(buffer[:n])
		if strings.HasPrefix(resolved, `\\?\UNC\`) {
			resolved = `\\` + strings.TrimPrefix(resolved, `\\?\UNC\`)
		} else {
			resolved = strings.TrimPrefix(resolved, `\\?\`)
		}
		return filepath.Clean(resolved), nil
	}
}
