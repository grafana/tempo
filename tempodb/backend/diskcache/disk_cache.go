package diskcache

import (
	"container/heap"
	"context"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/karrick/godirwalk"
)

// TODO: factor out common code with readOrCacheIndexToDisk into separate function
func (r *reader) readOrCacheBloom(ctx context.Context, blockID uuid.UUID, tenantID string, t string, shardNum int, miss bloomMissFunc) ([]byte, error, error) {
	var skippableError error

	k := key(blockID, tenantID, t)
	filename := path.Join(r.cfg.Path, k)

	bytes, err := ioutil.ReadFile(filename)

	if err != nil && !os.IsNotExist(err) {
		skippableError = err
	}

	if bytes != nil {
		return bytes, nil, nil
	}

	metricDiskCacheMiss.WithLabelValues(t).Inc()
	bytes, err = miss(ctx, blockID, tenantID, shardNum)
	if err != nil {
		return nil, nil, err // backend store error.  need to bubble this up
	}

	if bytes != nil {
		err = r.writeKeyToDisk(filename, bytes)
		if err != nil {
			skippableError = err
		}
	}

	return bytes, skippableError, nil
}

func (r *reader) readOrCacheIndex(ctx context.Context, blockID uuid.UUID, tenantID string, t string, miss indexMissFunc) ([]byte, error, error) {
	var skippableError error

	k := key(blockID, tenantID, t)
	filename := path.Join(r.cfg.Path, k)

	bytes, err := ioutil.ReadFile(filename)

	if err != nil && !os.IsNotExist(err) {
		skippableError = err
	}

	if bytes != nil {
		return bytes, nil, nil
	}

	metricDiskCacheMiss.WithLabelValues(t).Inc()
	bytes, err = miss(ctx, blockID, tenantID)
	if err != nil {
		return nil, nil, err // backend store error.  need to bubble this up
	}

	if bytes != nil {
		err = r.writeKeyToDisk(filename, bytes)
		if err != nil {
			skippableError = err
		}
	}

	return bytes, skippableError, nil
}

func (r *reader) writeKeyToDisk(filename string, b []byte) error {
	return ioutil.WriteFile(filename, b, 0644)
}

func (r *reader) startJanitor() {
	go func() {
		ticker := time.NewTicker(r.cfg.DiskCleanRate)
		for {
			select {
			case <-ticker.C:
				// repeatedly clean until we don't need to
				cleaned := true
				for cleaned {
					var err error
					cleaned, err = clean(r.cfg.Path, r.cfg.MaxDiskMBs, r.cfg.DiskPruneCount)
					if err != nil {
						metricDiskCacheClean.WithLabelValues("error").Inc()
						level.Error(r.logger).Log("msg", "error cleaning cache dir", "err", err)
					} else {
						metricDiskCacheClean.WithLabelValues("success").Inc()
					}
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

/* simplify */
func clean(folder string, allowedMBs int, pruneCount int) (bool, error) {

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

	if err != nil {
		return false, err
	}

	if totalSize < int64(allowedMBs*1024*1024) {
		return false, nil
	}

	// prune oldest files
	for fileInfoHeap.Len() > 0 {
		info := heap.Pop(&fileInfoHeap).(os.FileInfo)
		if info == nil {
			continue
		}

		err = os.Remove(path.Join(folder, info.Name()))
	}

	return true, err
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
		return AtimeNano(iStat) > AtimeNano(jStat)
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
