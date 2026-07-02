package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"swagger-annotator/internal/config"
	"swagger-annotator/internal/scanner"
	"swagger-annotator/internal/updater"
)

func main() {
	cfgPath := flag.String("config", "swagger_annotations.json", "Путь к файлу конфигурации")
	projectDir := flag.String("dir", ".", "Корневая директория проекта")
	dryRun := flag.Bool("dry-run", false, "Только показать изменения, не записывать в файлы")
	verbose := flag.Bool("verbose", false, "Подробный вывод")
	outputReport := flag.String("output", "", "Путь для сохранения отчёта")
	addDefaults := flag.Bool("add-defaults", false, "Добавлять дефолтные записи для новых хендлеров")
	skipExisting := flag.Bool("skip-existing", false, "Не обновлять существующие аннотации")
	useSemantic := flag.Bool("semantic", false, "Использовать семантический анализатор (медленнее, но точнее)")
	flag.Parse()

	log.SetFlags(0)
	if !*verbose {
		log.SetOutput(io.Discard) // подавляем детальный вывод
	} else {
		log.SetOutput(os.Stderr)
	}

	// Загрузка конфига
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка загрузки конфига: %v\n", err)
		os.Exit(1)
	}

	// Выбор сканера
	var finder scanner.HandlerFinder
	if *useSemantic {
		finder = &scanner.SemanticScanner{}
		fmt.Println("🔬 Используется семантический анализ (go/packages)")
	} else {
		finder = &scanner.ASTScanner{}
		fmt.Println("🔍 Используется синтаксический анализ (AST)")
	}

	// Поиск хендлеров
	handlers, err := finder.Find(*projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка анализа проекта: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📦 Найдено обработчиков: %d\n", len(handlers))

	// Сверка с конфигом: дополнение и актуализация
	added, matched := reconcileConfig(cfg, handlers, *addDefaults, *dryRun, *cfgPath)

	// Генерация/обновление аннотаций
	stats := processHandlers(cfg, handlers, matched, *skipExisting, *dryRun, *projectDir)

	// Вывод статистики и отчёта
	printStats(stats, added, len(handlers))
	if *outputReport != "" {
		if err := writeReport(*outputReport, stats, added, len(handlers), *projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Не удалось сохранить отчёт: %v\n", err)
		}
	}
}

func reconcileConfig(cfg *config.Config, handlers []scanner.HandlerLocation, addDefaults, dryRun bool, cfgPath string) ([]string, map[string]bool) {
	index := map[string]int{}
	for i, h := range cfg.Handlers {
		key := h.File + ":" + h.Function
		index[key] = i
	}

	matched := map[string]bool{}
	var added []string
	for _, h := range handlers {
		key := h.File + ":" + h.FuncName
		if _, exists := index[key]; exists {
			matched[key] = true
			continue
		}
		if addDefaults {
			cfg.Handlers = append(cfg.Handlers, config.HandlerDef{
				ID:       h.FuncName,
				File:     h.File,
				Function: h.FuncName,
			})
			added = append(added, key)
			log.Printf("➕ Добавлена запись по умолчанию для %s", key)
		} else {
			log.Printf("⚠️ Хендлер %s не найден в конфиге (используйте -add-defaults)", key)
		}
	}

	if len(added) > 0 && !dryRun {
		if err := config.Save(cfgPath, cfg); err != nil {
			log.Printf("⚠️ Ошибка сохранения обновлённого конфига: %v", err)
		} else {
			fmt.Println("✅ Файл конфигурации обновлён (добавлены новые хендлеры)")
		}
	} else if len(added) > 0 && dryRun {
		fmt.Println("🔍 dry-run: обновление конфига пропущено")
	}
	return added, matched
}

type Stats struct {
	Total   int
	Added   int
	Updated int
	Skipped int
	Errors  int
	Details []string
}

func processHandlers(cfg *config.Config, handlers []scanner.HandlerLocation, matched map[string]bool, skipExisting, dryRun bool, projectDir string) Stats {
	stats := Stats{Total: len(handlers)}
	cfgByKey := map[string]*config.HandlerDef{}
	for i := range cfg.Handlers {
		h := &cfg.Handlers[i]
		key := h.File + ":" + h.Function
		cfgByKey[key] = h
	}

	for _, hInfo := range handlers {
		key := hInfo.File + ":" + hInfo.FuncName
		def, exists := cfgByKey[key]
		if !exists || def.Summary == "" {
			stats.Skipped++
			continue
		}
		if skipExisting && hInfo.HasSwagger {
			stats.Skipped++
			log.Printf("⏭️ Пропущен (skip-existing): %s", key)
			continue
		}
		err := updater.UpdateComments(hInfo.File, hInfo.FuncName, hInfo.RecvType, def, dryRun, projectDir)
		if err != nil {
			stats.Errors++
			stats.Details = append(stats.Details, fmt.Sprintf("Ошибка %s: %v", key, err))
			fmt.Fprintf(os.Stderr, "❌ %s: %v\n", key, err)
			continue
		}
		if hInfo.HasSwagger {
			stats.Updated++
			log.Printf("✏️ Обновлён: %s", key)
		} else {
			stats.Added++
			log.Printf("✅ Добавлен: %s", key)
		}
	}
	return stats
}

func printStats(stats Stats, added []string, total int) {
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("        РЕЗУЛЬТАТ ОБРАБОТКИ")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Всего хендлеров в проекте: %d\n", total)
	fmt.Printf("Обработано:               %d\n", stats.Total)
	fmt.Printf("  Добавлено аннотаций:    %d\n", stats.Added)
	fmt.Printf("  Обновлено аннотаций:    %d\n", stats.Updated)
	fmt.Printf("  Пропущено:              %d\n", stats.Skipped)
	fmt.Printf("  Ошибок:                 %d\n", stats.Errors)
	if len(added) > 0 {
		fmt.Printf("Новых записей в конфиге:  %d\n", len(added))
	}
	if len(stats.Details) > 0 {
		fmt.Println("\nДетали ошибок:")
		for _, d := range stats.Details {
			fmt.Println("  -", d)
		}
	}
}

func writeReport(path string, stats Stats, added []string, total int, projectDir string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "Отчёт swagger-annotator\n")
	fmt.Fprintf(f, "Дата: %s\n", time.Now().Format("02.01.2006 15:04:05"))
	fmt.Fprintf(f, "Проект: %s\n\n", projectDir)
	fmt.Fprintf(f, "Всего хендлеров: %d\n", total)
	fmt.Fprintf(f, "Добавлено аннотаций: %d\n", stats.Added)
	fmt.Fprintf(f, "Обновлено: %d\n", stats.Updated)
	fmt.Fprintf(f, "Пропущено: %d\n", stats.Skipped)
	fmt.Fprintf(f, "Ошибок: %d\n", stats.Errors)
	if len(added) > 0 {
		fmt.Fprintf(f, "Новых записей в конфиге: %d\n", len(added))
		for _, a := range added {
			fmt.Fprintf(f, "  - %s\n", a)
		}
	}
	if len(stats.Details) > 0 {
		fmt.Fprintf(f, "\nОшибки:\n")
		for _, d := range stats.Details {
			fmt.Fprintf(f, "  - %s\n", d)
		}
	}
	return nil
}
