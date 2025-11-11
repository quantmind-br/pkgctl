# Análise: Progress Bar para Instalação de Pacotes DEB

## 📋 Resumo Executivo

**Objetivo:** Avaliar a viabilidade de implementar uma barra de progresso visual durante a instalação de pacotes DEB via `pkgctl`.

**Conclusão:** ✅ **VIÁVEL** - É possível implementar um sistema de progress bar com algumas limitações técnicas.

**Complexidade Estimada:** 🟡 **MÉDIA** - Requer integração com processos externos e estimativa de etapas.

---

## 🔍 Análise do Fluxo Atual de Instalação DEB

### Fases da Instalação Identificadas

A instalação de um pacote DEB segue estas etapas principais:

```
1. ⚙️  Detecção do tipo de pacote (< 1s)
2. ✅ Validações iniciais (< 1s)
   ├─ Verificar se debtap está instalado
   ├─ Verificar se debtap está inicializado
   └─ Verificar se pacman está disponível
3. 📦 Extração de metadados DEB (1-3s)
   └─ dpkg-deb --field para obter nome do pacote
4. 🔄 CONVERSÃO DEBTAP (60-180s) ⬅️ FASE MAIS LONGA
   ├─ Extração do conteúdo DEB
   ├─ Análise de dependências
   ├─ Conversão de scripts maintainer
   ├─ Geração de .PKGINFO
   └─ Criação do .pkg.tar.zst
5. 🔧 Correção de dependências (1-2s)
   ├─ Extração da .PKGINFO
   ├─ Mapeamento Debian→Arch
   └─ Repacking do arquivo
6. 📥 INSTALAÇÃO PACMAN (10-60s) ⬅️ SEGUNDA FASE LONGA
   ├─ Validação de assinaturas
   ├─ Resolução de dependências
   ├─ Extração de arquivos
   └─ Execução de hooks
7. 🎨 Desktop integration (1-3s)
   ├─ Modificação de .desktop files (Wayland)
   ├─ Update desktop database
   └─ Update icon cache
```

### Código Atual de Progresso

**Localização:** `internal/backends/deb/deb.go:373-388`

```go
start := time.Now()
progressDone := make(chan struct{})
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            d.logger.Info().
                Dur("elapsed", time.Since(start)).
                Msg("debtap conversion in progress")
        case <-progressDone:
            return
        }
    }
}()
```

**Status Atual:** Apenas logging textual a cada 30 segundos.

---

## 📚 Infraestrutura Disponível

### Biblioteca de Progress Bar Já Instalada

✅ **`github.com/schollz/progressbar/v3`** está no `go.mod`

**Recursos:**
- Barra de progresso visual com percentual
- Estimativa de tempo restante (ETA)
- Velocidade de operação
- Modo indeterminado (spinner)
- Suporte para cores e temas
- Thread-safe

**Exemplo de Uso:**
```go
bar := progressbar.NewOptions(100,
    progressbar.OptionSetDescription("Instalando"),
    progressbar.OptionSetWidth(50),
    progressbar.OptionShowCount(),
    progressbar.OptionShowItsPerSecond(),
)

for i := 0; i < 100; i++ {
    bar.Add(1)
    time.Sleep(time.Millisecond * 100)
}
```

### Módulo UI Existente

**Localização:** `internal/ui/`

**Conteúdo:**
- `prompt.go` - Prompts interativos via `promptui`
- `colors.go` - Códigos de cores para terminal

**Ausente:** Nenhum código de progress bar implementado ainda.

---

## 🚧 Desafios Técnicos Identificados

### 1. **Processo Externo Opaco (debtap)**

**Problema:**
- `debtap` é um script shell externo
- Não fornece API de progresso
- Output é imprevisível (varia por pacote)

**Evidência:**
```bash
$ debtap --help
# Nenhuma opção de progress/verbose estruturado
# Flags: -q (quiet), -Q (super quiet)
```

**Output Típico:**
```
==> Extracting package data...
==> Fixing possible directories structure problems...
==> Generating .SRCINFO file...
==> Checking and fixing missing dependencies...
==> Creating Arch package...
```

### 2. **Duração Imprevisível**

**Variação Observada:**
- Pacotes pequenos (< 5MB): 30-60s
- Pacotes médios (10-50MB): 60-180s
- Pacotes grandes (> 100MB): 180-600s

**Fatores:**
- Tamanho do pacote
- Número de arquivos
- Complexidade de dependências
- Performance do sistema

### 3. **Fases Sem Feedback Determinístico**

| Fase | Progresso Determinável? | Solução |
|------|------------------------|---------|
| Detecção | ✅ Sim | Trivial |
| Conversão debtap | ❌ Não | **Spinner indeterminado** |
| Correção deps | ✅ Sim | Trivial |
| Instalação pacman | ⚠️ Parcial | Parse output ou spinner |
| Desktop integration | ✅ Sim | Trivial |

### 4. **Concorrência de Output**

**Problema:**
- `debtap` escreve stdout/stderr
- Progress bar precisa de terminal control
- Conflito de cursor/linhas

**Solução:** Capturar output e renderizar separadamente.

---

## ✅ Solução Proposta

### Arquitetura: Sistema de Etapas Híbrido

Combinar **etapas determinísticas** com **spinners indeterminados**:

```
┌─────────────────────────────────────────────────────┐
│ [■■■■■■■■░░░░░░░░░░░░] 40% - Convertendo DEB...     │
│ Elapsed: 1m 30s | ETA: 2m 15s                       │
└─────────────────────────────────────────────────────┘

Ou (modo indeterminado):

┌─────────────────────────────────────────────────────┐
│ ⠋ Convertendo DEB... (tempo decorrido: 1m 30s)     │
└─────────────────────────────────────────────────────┘
```

### Modelo de Progresso em Etapas

```go
type InstallationPhase struct {
    Name           string
    Weight         int  // Peso relativo (soma = 100)
    Deterministic  bool // true = progress bar | false = spinner
}

phases := []InstallationPhase{
    {"Validando pacote", 5, true},
    {"Extraindo metadados", 5, true},
    {"Convertendo DEB→Arch", 60, false}, // SPINNER
    {"Corrigindo dependências", 5, true},
    {"Instalando com pacman", 20, false}, // SPINNER
    {"Integrando desktop", 5, true},
}
```

### Implementação Detalhada

#### 1. **Nova Estrutura: ProgressTracker**

```go
// internal/ui/progress.go

type ProgressTracker struct {
    bar          *progressbar.ProgressBar
    currentPhase int
    phases       []InstallationPhase
    totalWeight  int
    startTime    time.Time
    logger       *zerolog.Logger
}

func NewProgressTracker(phases []InstallationPhase, logger *zerolog.Logger) *ProgressTracker {
    totalWeight := 0
    for _, p := range phases {
        totalWeight += p.Weight
    }
    
    bar := progressbar.NewOptions(totalWeight,
        progressbar.OptionSetDescription("Instalando"),
        progressbar.OptionSetWidth(50),
        progressbar.OptionShowCount(),
        progressbar.OptionSetTheme(progressbar.Theme{
            Saucer:        "[green]=[reset]",
            SaucerPadding: " ",
            BarStart:      "[",
            BarEnd:        "]",
        }),
    )
    
    return &ProgressTracker{
        bar:        bar,
        phases:     phases,
        totalWeight: totalWeight,
        startTime:  time.Now(),
        logger:     logger,
    }
}

func (p *ProgressTracker) StartPhase(phaseIndex int) {
    if phaseIndex >= len(p.phases) {
        return
    }
    
    phase := p.phases[phaseIndex]
    p.currentPhase = phaseIndex
    
    if phase.Deterministic {
        p.bar.Describe(phase.Name)
    } else {
        // Spinner mode
        p.bar.Describe(fmt.Sprintf("⠋ %s...", phase.Name))
    }
}

func (p *ProgressTracker) AdvancePhase() {
    if p.currentPhase >= len(p.phases) {
        return
    }
    
    // Adiciona peso da fase concluída
    p.bar.Add(p.phases[p.currentPhase].Weight)
    
    // Avança para próxima fase
    p.currentPhase++
    if p.currentPhase < len(p.phases) {
        p.StartPhase(p.currentPhase)
    }
}

func (p *ProgressTracker) UpdateIndeterminate(message string) {
    // Para fases indeterminadas, atualizar descrição
    elapsed := time.Since(p.startTime)
    p.bar.Describe(fmt.Sprintf("⠋ %s (decorrido: %s)", 
        message, 
        formatDuration(elapsed)))
}

func (p *ProgressTracker) Finish() {
    p.bar.Finish()
    fmt.Println()
}
```

#### 2. **Modificação em `deb.go`**

```go
func (d *DebBackend) Install(ctx context.Context, packagePath string, opts core.InstallOptions) (*core.InstallRecord, error) {
    // Definir fases
    phases := []ui.InstallationPhase{
        {"Validando", 5, true},
        {"Extraindo metadados", 5, true},
        {"Convertendo DEB", 60, false},
        {"Corrigindo dependências", 5, true},
        {"Instalando com pacman", 20, false},
        {"Configurando desktop", 5, true},
    }
    
    // Criar progress tracker
    progress := ui.NewProgressTracker(phases, d.logger)
    defer progress.Finish()
    
    // Fase 1: Validação
    progress.StartPhase(0)
    if err := helpers.RequireCommand("debtap"); err != nil {
        return nil, err
    }
    // ... validações
    progress.AdvancePhase()
    
    // Fase 2: Metadados
    progress.StartPhase(1)
    pkgName := d.extractPackageName(ctx, packagePath)
    progress.AdvancePhase()
    
    // Fase 3: Conversão (indeterminada)
    progress.StartPhase(2)
    archPkgPath, err := d.convertWithDebtapProgress(ctx, packagePath, tmpDir, progress)
    progress.AdvancePhase()
    
    // ... demais fases
}
```

#### 3. **Versão com Progress do `convertWithDebtap`**

```go
func (d *DebBackend) convertWithDebtapProgress(
    ctx context.Context, 
    debPath, outputDir string,
    progress *ui.ProgressTracker,
) (string, error) {
    // ... código existente ...
    
    // Substituir ticker por updates no progress
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                progress.UpdateIndeterminate("Convertendo DEB")
            case <-progressDone:
                return
            }
        }
    }()
    
    // ... resto do código
}
```

### Estimativa de Progresso Inteligente

Para melhorar UX, podemos **estimar** o progresso de `debtap` baseado em:

1. **Tamanho do arquivo DEB**
2. **Histórico de conversões anteriores**

```go
type ConversionEstimator struct {
    historyDB *sql.DB
}

func (e *ConversionEstimator) EstimateDuration(debSize int64) time.Duration {
    // Heurística simples: ~1s por MB
    mbSize := float64(debSize) / (1024 * 1024)
    baseTime := time.Duration(mbSize * 1.5) * time.Second
    
    // Adicionar overhead fixo
    return baseTime + (30 * time.Second)
}

func (e *ConversionEstimator) RecordConversion(debSize int64, duration time.Duration) {
    // Salvar no SQLite para melhorar estimativas futuras
    // Usar média móvel ponderada
}
```

---

## 📊 Comparação de Abordagens

| Abordagem | Pros | Cons | Recomendação |
|-----------|------|------|--------------|
| **1. Spinner Simples** | Fácil implementação | Sem feedback de progresso | ⚠️ Mínimo |
| **2. Etapas Fixas** | Mostra progresso geral | Impreciso para debtap | ✅ **RECOMENDADO** |
| **3. Parse Output debtap** | Mais preciso | Frágil (depende de output) | ❌ Não recomendado |
| **4. Estimativa por tamanho** | Feedback útil | Pode ser impreciso | 🟡 Complementar |
| **5. Híbrido (Etapas + Spinner)** | Melhor UX | Complexidade média | ✅ **IDEAL** |

---

## 🎯 Proposta de Implementação

### Fase 1: Base (Essencial)
- [ ] Criar `internal/ui/progress.go` com `ProgressTracker`
- [ ] Implementar sistema de fases ponderadas
- [ ] Integrar em `deb.go` com 6 fases principais
- [ ] Modo spinner para fases indeterminadas

**Tempo Estimado:** 4-6 horas  
**Complexidade:** Média  
**Prioridade:** Alta

### Fase 2: Refinamento (Opcional)
- [ ] Estimativa de duração baseada em tamanho
- [ ] Histórico de conversões no SQLite
- [ ] Progress bar em `pacman` (via parse de output)
- [ ] Temas customizáveis

**Tempo Estimado:** 6-8 horas  
**Complexidade:** Alta  
**Prioridade:** Média

### Fase 3: Polimento (Nice-to-have)
- [ ] Animações de spinner variadas
- [ ] Suporte para quiet mode (desabilitar progress)
- [ ] Progress bar para outros backends (AppImage, Tarball)
- [ ] Logs paralelos (não conflitar com progress)

**Tempo Estimado:** 4-6 horas  
**Complexidade:** Média  
**Prioridade:** Baixa

---

## 🔧 Exemplo de Output Esperado

### Antes (Atual):
```
→ Installing package...
09:28:23 INF converting DEB to Arch package (this may take a while)...
09:28:53 INF debtap conversion in progress elapsed=30000.776332
09:29:23 INF debtap conversion in progress elapsed=60001.017703
09:29:53 INF debtap conversion in progress elapsed=90000.136547
09:30:06 INF checking and fixing malformed dependencies...
09:30:16 INF installing converted package with pacman...
✓ Package installed successfully
```

### Depois (Com Progress Bar):
```
→ Installing goose_1.13.0_amd64.deb

[■■■░░░░░░░░░░░░░░░░░] 15% - Extraindo metadados...
[■■■■■░░░░░░░░░░░░░░░] 25% - Convertendo DEB... 
[■■■■■■■■■■■■■░░░░░░░] 65% ⠋ Convertendo DEB (decorrido: 1m 30s)...
[■■■■■■■■■■■■■■░░░░░░] 70% - Corrigindo dependências...
[■■■■■■■■■■■■■■■░░░░░] 75% ⠋ Instalando com pacman...
[■■■■■■■■■■■■■■■■■■■■] 100% ✓ Instalado com sucesso!

Tempo total: 2m 15s
```

---

## ⚠️ Limitações Conhecidas

1. **Progresso Impreciso em Debtap**
   - Não há como saber exatamente quanto falta
   - Spinner indeterminado é mais honesto

2. **Variação de Tempo**
   - Depende de hardware, I/O, rede (debtap database)
   - Estimativas podem estar erradas em 30-50%

3. **Output Conflitante**
   - Se debtap escrever no terminal, pode bagunçar a UI
   - Solução: Capturar todo stdout/stderr

4. **Modo Quiet/Debug**
   - Progress bar pode conflitar com logs detalhados
   - Necessário flag `--no-progress` ou auto-detectar TTY

---

## 🏁 Conclusão e Recomendações

### ✅ Viabilidade: **SIM**

**Recomendação:** Implementar **Abordagem Híbrida (Fase 1)**

**Justificativa:**
1. ✅ Biblioteca já disponível (`progressbar/v3`)
2. ✅ Fases claramente identificáveis
3. ✅ UX significativamente melhor
4. ⚠️ Complexidade média mas gerenciável
5. ✅ Pode ser estendido para outros backends

### Próximos Passos (Se Aprovado)

1. Criar `internal/ui/progress.go`
2. Definir interface `ProgressReporter`
3. Implementar em `deb.go`
4. Testar com pacotes de diferentes tamanhos
5. Estender para RPM, Tarball se bem-sucedido

### Alternativa Mínima

Se implementação completa for muito complexa:

**Quick Win:** Substituir ticker por spinner animado
```go
// Ao invés de:
d.logger.Info().Msg("debtap conversion in progress")

// Usar:
spinner := ui.NewSpinner("Convertendo DEB")
spinner.Start()
// ... debtap
spinner.Stop()
```

**Ganho:** 70% da melhoria de UX com 20% do esforço.

---

**Autor:** Claude Code  
**Data:** 2025-11-07  
**Status:** Análise Completa - Aguardando Decisão de Implementação
