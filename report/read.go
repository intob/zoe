package report

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/swissinfo-ch/zoe/ev"
	"google.golang.org/protobuf/proto"
)

func (r *Runner) readEventsFromFile() {
	// Reset the event count
	r.currentReportEventCount = 0

	// Open the file
	file, err := os.Open(r.filename)
	if err != nil {
		panic(fmt.Sprintf("failed to open file for reading: %v", err))
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	fileSize := fileInfo.Size()
	r.fileSize = fileSize

	// Starting from the end of the file, read backwards
	for fileSize > 0 {
		// Move back to read the four-byte length at the end of the compressed event block
		offset := fileSize - 4
		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the four-byte length, big endian
		lengthBytes := make([]byte, 4)
		_, err = file.Read(lengthBytes)
		if err != nil {
			panic(fmt.Sprintf("failed to read event size: %v", err))
		}
		length := (int(lengthBytes[0]) << 24) + (int(lengthBytes[1]) << 16) + (int(lengthBytes[2]) << 8) + int(lengthBytes[3])

		// Validate length and ensure offset does not go beyond the file start
		if length <= 0 || length > int(fileSize-4) {
			fmt.Println("invalid block length or corrupted file")
			break
		}

		// Move back to read the compressed block payload
		offset -= int64(length)
		if offset < 0 {
			panic("offset calculated is beyond the file start, indicating a potential error in block length or file corruption")
		}
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			panic(fmt.Sprintf("failed to seek in file: %v", err))
		}

		// Read the compressed block payload
		compressedData := make([]byte, length)
		_, err = file.Read(compressedData)
		if err != nil {
			panic(fmt.Sprintf("failed to read compressed event payload: %v", err))
		}

		// Decompress the block payload
		gzr, err := gzip.NewReader(bytes.NewBuffer(compressedData))
		if err != nil {
			panic(fmt.Sprintf("failed to create gzip reader: %v", err))
		}
		decompressedData, err := io.ReadAll(gzr)
		if err != nil {
			panic(fmt.Sprintf("failed to decompress event payload: %v", err))
		}
		gzr.Close()

		// Unmarshal the block
		block := &ev.Block{}
		if err := proto.Unmarshal(decompressedData, block); err != nil {
			panic(fmt.Sprintf("failed to unmarshal block: %v", err))
		}

		// Send the events to the channel
		for _, e := range block.GetEvs() {
			r.events <- e
		}

		// Increment the event count
		r.currentReportEventCount += uint32(len(block.GetEvs()))

		// Update fileSize to the new offset for the next iteration
		fileSize = offset
	}

	// Close the events channel after reading all events
	close(r.events)

	// Update the last report event count
	r.lastReportEventCount = r.currentReportEventCount
}
