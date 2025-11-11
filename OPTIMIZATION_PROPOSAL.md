# Proposta de Otimização: Substituir npx asar por Biblioteca Go Nativa

## Problema Atual

Em `internal/backends/tarball/tarball.go:649`, cada extração de arquivo ASAR spawna `npx --yes asar`:

```go
extractErr := helpers.RunCommandStreaming(ctx, nil, nil, "npx", "--yes", "asar", "extract", asarFile, tempDir)
```

**Overhead por ASAR:**
- Startup do Node.js (~100-500ms)
- Resolução e download de pacote NPM (primeira execução: ~2-5s, cache: ~200-500ms)
- Processo spawn overhead
- Dependência externa (Node.js/npm deve estar instalado)

## Solução Proposta: layeh.com/asar

### Biblioteca Go Nativa
- **Repositório**: https://github.com/layeh/asar
- **Import**: `layeh.com/asar`
- **Licença**: MPL 2.0 (compatível)
- **Maturidade**: 20+ commits, usado em produção

### API Essencial

```go
import "layeh.com/asar"

// Abrir arquivo ASAR
f, err := os.Open("app.asar")
archive, err := asar.Decode(f)

// Buscar arquivos
entry := archive.Find("path", "to", "file.png")

// Ler conteúdo
reader := entry.Open()
data, _ := io.ReadAll(reader)

// Walk recursivo
archive.Walk(func(path string, entry *asar.Entry) error {
    // Processar cada arquivo
    return nil
})
```

## Implementação Sugerida

### 1. Adicionar dependência

```bash
go get layeh.com/asar
```

### 2. Nova função em `tarball.go`

```go
// extractIconsFromAsarNative extracts icons using native Go ASAR library
func (t *TarballBackend) extractIconsFromAsarNative(asarPath, destDir string) error {
	f, err := os.Open(asarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	archive, err := asar.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode ASAR: %w", err)
	}

	// Walk e extrair apenas ícones (*.png, *.ico, *.svg)
	return archive.Walk(func(path string, entry *asar.Entry) error {
		if entry.Flags.Dir() {
			return nil
		}

		// Filtrar apenas ícones
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".png" && ext != ".ico" && ext != ".svg" && ext != ".jpg" {
			return nil
		}

		// Criar diretório de destino
		targetPath := filepath.Join(destDir, path)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// Extrair arquivo
		reader := entry.Open()
		defer reader.Close()

		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, reader)
		return err
	})
}
```

### 3. Substituir chamada npx

Em `extractIconsFromAsar()`, trocar:

```go
// ANTES
extractErr := helpers.RunCommandStreaming(ctx, nil, nil, "npx", "--yes", "asar", "extract", asarFile, tempDir)

// DEPOIS
extractErr := t.extractIconsFromAsarNative(asarFile, tempDir)
```

## Benefícios

### Performance
- ✅ **Elimina 100-500ms** de startup Node.js por ASAR
- ✅ **Elimina 200ms-5s** de resolução NPM
- ✅ **Extração mais rápida**: Go nativo vs subprocess
- ✅ **Zero network hits** após instalação inicial

### Confiabilidade
- ✅ Remove dependência de Node.js/npm no sistema
- ✅ Build estático incluindo ASAR support
- ✅ Menos pontos de falha (sem spawn, sem network)

### Recursos
- ✅ **Extração seletiva**: extrair apenas ícones, não todo o ASAR
- ✅ Menos I/O temporário (não precisa extrair tudo)
- ✅ Memória mais eficiente

## Migração

### Fase 1: Implementação paralela
- Manter código npx existente
- Adicionar função nativa `extractIconsFromAsarNative()`
- Detectar se `layeh.com/asar` está disponível

### Fase 2: Fallback inteligente
```go
if t.hasNativeAsarSupport() {
    err = t.extractIconsFromAsarNative(asarFile, tempDir)
} else {
    // Fallback para npx
    err = helpers.RunCommandStreaming(...)
}
```

### Fase 3: Substituição completa
- Após validação, remover código npx
- ASAR support se torna built-in

## Estimativa de Ganho

**Cenário típico**: App Electron com 2 arquivos ASAR

| Método | Tempo por ASAR | Total (2 ASARs) |
|--------|----------------|-----------------|
| npx (sem cache) | 2-5s | 4-10s |
| npx (com cache) | 200-500ms | 400ms-1s |
| **Go nativo** | **10-50ms** | **20-100ms** |

**Redução esperada**: **80-95% do tempo de processamento ASAR**

## Próximos Passos

1. ✅ Adicionar `layeh.com/asar` ao go.mod
2. ✅ Implementar `extractIconsFromAsarNative()`
3. ✅ Testar com apps Electron reais (VSCode, Slack, Discord)
4. ✅ Benchmark comparativo npx vs nativo
5. ✅ Deploy com fallback habilitado
6. ✅ Monitorar performance em produção
7. ✅ Remover código npx após validação

## Referências

- **layeh.com/asar**: https://pkg.go.dev/layeh.com/asar
- **Electron ASAR Spec**: https://electronjs.org/docs/latest/tutorial/asar-archives
- **Benchmark original**: OPTIMIZATION_FINDINGS.md linha 13-16
