package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptionKey() []byte {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return buf
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) // 16 bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// prepend the iv to the file
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	buf := make([]byte, 32*1024)
	stream := cipher.NewCTR(block, iv)
	nw := block.BlockSize()

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nb, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nb
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, err
		}
	}

	return nw, err
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	buf := make([]byte, 32*1024)
	stream := cipher.NewCTR(block, iv)
	nw := block.BlockSize()

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nb, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nb
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, nil
		}
	}

	return nw, nil
}
