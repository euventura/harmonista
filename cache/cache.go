package cache

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const baseCacheDir = "./cache"

// GenerateCacheKey cria uma chave de cache baseada no subdomínio e path
func GenerateCacheKey(subdomain, path string) string {
	// Normalizar o path
	if path == "" || path == "/" {
		path = "index"
	}

	// Remover barra inicial
	path = strings.TrimPrefix(path, "/")

	// Substituir caracteres problemáticos para nomes de arquivo
	path = strings.ReplaceAll(path, "/", "_")

	// Criar hash para evitar nomes muito longos
	hash := sha256.Sum256([]byte(path))
	filename := fmt.Sprintf("%s_%x.html", path, hash[:8])

	return filename
}

// GetCachePath retorna o caminho completo do arquivo de cache
func GetCachePath(subdomain, path string) string {
	blogCacheDir := filepath.Join(baseCacheDir, subdomain)
	filename := GenerateCacheKey(subdomain, path)
	return filepath.Join(blogCacheDir, filename)
}

// GetCacheDir retorna o diretório de cache do blog
func GetCacheDir(subdomain string) string {
	return filepath.Join(baseCacheDir, subdomain)
}

// Exists verifica se o cache existe
func Exists(subdomain, path string) bool {
	cachePath := GetCachePath(subdomain, path)
	_, err := os.Stat(cachePath)
	return err == nil
}

// Read lê o conteúdo do cache
func Read(subdomain, path string) ([]byte, error) {
	cachePath := GetCachePath(subdomain, path)
	return ioutil.ReadFile(cachePath)
}

// Write salva o conteúdo no cache
func Write(subdomain, path string, content []byte) error {
	blogCacheDir := GetCacheDir(subdomain)

	// Criar diretório do blog se não existir
	if err := os.MkdirAll(blogCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := GetCachePath(subdomain, path)

	// Salvar o arquivo
	if err := ioutil.WriteFile(cachePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	log.Printf("Cache written: %s", cachePath)
	return nil
}

// Delete deleta um arquivo de cache específico
func Delete(subdomain, path string) error {
	cachePath := GetCachePath(subdomain, path)

	if !Exists(subdomain, path) {
		// Arquivo não existe, não é um erro
		return nil
	}

	if err := os.Remove(cachePath); err != nil {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	log.Printf("Cache deleted: %s", cachePath)
	return nil
}

// DeleteAll deleta todo o cache de um blog
func DeleteAll(subdomain string) error {
	blogCacheDir := GetCacheDir(subdomain)

	// Verificar se o diretório existe
	if _, err := os.Stat(blogCacheDir); os.IsNotExist(err) {
		// Diretório não existe, não é um erro
		return nil
	}

	// Remover todo o diretório do blog
	if err := os.RemoveAll(blogCacheDir); err != nil {
		return fmt.Errorf("failed to delete blog cache: %w", err)
	}

	log.Printf("All cache deleted for blog: %s", subdomain)
	return nil
}

// DeletePostCache deleta o cache de um post específico
func DeletePostCache(subdomain, postSlug string) error {
	// Deletar cache do post individual
	if err := Delete(subdomain, "/"+postSlug); err != nil {
		return err
	}

	// Deletar também o cache da página inicial (já que lista os posts)
	if err := Delete(subdomain, "/"); err != nil {
		return err
	}

	log.Printf("Post cache deleted: %s/%s", subdomain, postSlug)
	return nil
}

// DeletePageCache deleta o cache de uma página específica
func DeletePageCache(subdomain, pageSlug string) error {
	// Deletar cache da página individual
	return Delete(subdomain, "/p/"+pageSlug)
}
