package cache

import (
	"container/heap"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/karrick/godirwalk"
)

func (r *reader) readOrCacheKeyToDisk(blockID uuid.UUID, tenantID string, t string, miss func(blockID uuid.UUID, tenantID string) ([]byte, error)) ([]byte, error) {
	r.lock.RLock()

	k := key(blockID, tenantID, t)
	filename := path.Join(r.cfg.Path, k)

	bytes, err := ioutil.ReadFile(filename)

	if err != nil && !os.IsNotExist(err) {
		r.lock.RUnlock()
		return nil, err // todo: just ignore this error and go to the backing store?
	}

	if bytes != nil {
		r.lock.RUnlock()
		return bytes, nil
	}

	r.lock.RUnlock()
	bytes, err = miss(blockID, tenantID)
	if err != nil {
		return nil, err
	}

	if bytes != nil {
		err = r.writeKeyToDisk(filename, bytes)
		if err != nil {
			return nil, err // jpe: ignore this error?
		}
	}

	return bytes, nil
}

func (r *reader) writeKeyToDisk(filename string, b []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	return ioutil.WriteFile(filename, b, 0644)
}

func (r *reader) startJanitor() {
	go func() {
		ticker := time.NewTicker(r.cfg.DiskCleanRate)
		for {
			select {
			case <-ticker.C:
				// repeatedly clean until we don't need to
				for clean(r.cfg.Path, r.cfg.MaxDiskMBs, r.cfg.DiskPruneCount) {
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

/* simplify */
func clean(folder string, allowedMBs int, pruneCount int) bool {

	var totalSize int64
	fileInfoHeap := FileInfoHeap(make([]os.FileInfo, 0, pruneCount))
	heap.Init(&fileInfoHeap)

	err := godirwalk.Walk(folder, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				return nil
			}

			info, err := os.Stat(osPathname)
			if err != nil {
				return err
			}

			totalSize += info.Size()

			for len(fileInfoHeap) >= cap(fileInfoHeap) {
				heap.Pop(&fileInfoHeap)
			}
			heap.Push(&fileInfoHeap, info)
			return nil
		},
		Unsorted: true,
	})

	// jpe : err?
	if err != nil {
		return false
	}

	if totalSize < int64(allowedMBs*1024*1024) {
		return false
	}

	// prune oldest files
	for fileInfoHeap.Len() > 0 {
		info := heap.Pop(&fileInfoHeap).(os.FileInfo)
		if info == nil {
			continue
		}

		os.Remove(path.Join(folder, info.Name()))
	}

	return true
}

type FileInfoHeap []os.FileInfo

func (h FileInfoHeap) Len() int {
	return len(h)
}

func (h FileInfoHeap) Less(i, j int) bool {
	iInfo := h[i]
	jInfo := h[j]

	if iInfo == nil {
		return false
	}
	if jInfo == nil {
		return true
	}

	iStat, iOK := iInfo.Sys().(*syscall.Stat_t)
	jStat, jOK := jInfo.Sys().(*syscall.Stat_t)

	if iOK && jOK {
		return iStat.Atim.Nano() > jStat.Atim.Nano()
	}

	return iInfo.ModTime().After(jInfo.ModTime())
}

func (h FileInfoHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *FileInfoHeap) Push(x interface{}) {
	item := x.(os.FileInfo)
	*h = append(*h, item)
}

func (h *FileInfoHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[0 : n-1]
	return item
}
