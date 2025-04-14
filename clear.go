package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	// main - точка входа в программу.
	// Обрабатывает флаги командной строки, читает входной HTML файл,
	// выполняет выбранное действие (очистка, разделение, удаление CSS, или все),
	// и сохраняет результаты в выходной каталог.
	filePath := flag.String("file", "", "Путь к входному HTML файлу (обязательно)")
	outputDir := flag.String("output-dir", "", "Каталог для сохранения результатов (по умолчанию: каталог входного файла)")
	action := flag.String("action", "", "Действие: 'clean' (очистить), 'split' (разделить), 'remove-css' (удалить неиспользуемый CSS и разделить), 'all' (все действия)")

	removeUnusedCSSFlag := flag.Bool("remove-unused-css", false, "(старый флаг) Удалять неиспользуемые CSS стили и разделить")
	splitFilesFlag := flag.Bool("split", false, "(старый флаг) Разделить код на HTML, CSS и JS файлы")
	cleanCodeFlag := flag.Bool("clean", false, "(старый флаг) Очистить код от мусора (base64, комментарии и т.д.)")

	flag.Parse()

	if *filePath == "" {
		fmt.Println("Ошибка: Пожалуйста, укажите путь к файлу с помощью флага -file")
		flag.Usage()
		os.Exit(1)
	}

	chosenAction := *action
	// Определение действия на основе новых и старых флагов.
	if chosenAction == "" {
		if *removeUnusedCSSFlag {
			chosenAction = "remove-css"
		} else if *splitFilesFlag {
			chosenAction = "split"
		} else if *cleanCodeFlag {
			chosenAction = "clean"
		} else {
			fmt.Println("Ошибка: Пожалуйста, укажите действие с помощью флага -action ('clean', 'split', 'remove-css', 'all')")
			fmt.Println("Или используйте старые флаги: -clean, -split, -remove-unused-css")
			flag.Usage()
			os.Exit(1)
		}
		fmt.Println("Предупреждение: Используются старые флаги. Рекомендуется использовать флаг -action.")
	}

	// Установка флагов в зависимости от выбранного действия 'action'.
	if chosenAction == "all" {
		*cleanCodeFlag = true
		*splitFilesFlag = true
		*removeUnusedCSSFlag = true
	} else if chosenAction == "remove-css" {
		*cleanCodeFlag = true
		*splitFilesFlag = true
		*removeUnusedCSSFlag = true
	} else if chosenAction == "split" {
		*cleanCodeFlag = true
		*splitFilesFlag = true
		*removeUnusedCSSFlag = false
	} else if chosenAction == "clean" {
		*cleanCodeFlag = true
		*splitFilesFlag = false
		*removeUnusedCSSFlag = false
	} else if chosenAction != "" && chosenAction != "clean" && chosenAction != "split" && chosenAction != "remove-css" && chosenAction != "all" {
		fmt.Printf("Ошибка: Неизвестное действие: %s\n", chosenAction)
		flag.Usage()
		os.Exit(1)
	}

	htmlContentBytes, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Printf("Ошибка при чтении файла %s: %v\n", *filePath, err)
		os.Exit(1)
	}
	htmlContent := string(htmlContentBytes)

	outDir := *outputDir
	// Если выходной каталог не указан, используется каталог входного файла.
	if outDir == "" {
		outDir = filepath.Dir(*filePath)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Ошибка при создании выходного каталога %s: %v\n", outDir, err)
		os.Exit(1)
	}

	baseName := strings.TrimSuffix(filepath.Base(*filePath), filepath.Ext(*filePath))
	outputHTMLPath := filepath.Join(outDir, baseName+".html")
	outputCSSPath := filepath.Join(outDir, baseName+".css")
	outputJSPath := filepath.Join(outDir, baseName+".js")
	cssFileName := filepath.Base(outputCSSPath)
	jsFileName := filepath.Base(outputJSPath)

	var (
		processedHTML string
		extractedCSS  string
		extractedJS   string
	)

	processedHTML = htmlContent

	// Выполнение очистки кода, если установлен флаг cleanCodeFlag.
	if *cleanCodeFlag {
		fmt.Println("Очистка кода...")
		processedHTML = cleanCode(processedHTML)
		fmt.Println("Очистка завершена.")
	}

	// Выполнение разделения контента и обработки CSS, если установлен флаг splitFilesFlag.
	if *splitFilesFlag {
		fmt.Println("Разделение контента...")
		modifiedHTML, tempCSS, tempJS := separateContent(processedHTML)
		processedHTML = modifiedHTML
		extractedCSS = tempCSS
		extractedJS = tempJS
		fmt.Println("Разделение завершено.")

		// Удаление неиспользуемого CSS, если установлен флаг removeUnusedCSSFlag и CSS был извлечен.
		if *removeUnusedCSSFlag && extractedCSS != "" {
			fmt.Println("Удаление неиспользуемого CSS...")
			extractedCSS = removeUnusedCSS(extractedCSS, processedHTML)
			fmt.Println("Удаление неиспользуемого CSS завершено.")
		}

		fmt.Println("Добавление ссылок на CSS/JS в HTML...")
		processedHTML = addLinksToHTML(processedHTML, extractedCSS, extractedJS, cssFileName, jsFileName)

		fmt.Printf("Сохранение файлов в каталог: %s\n", outDir)
		if err := saveToFile(outputHTMLPath, processedHTML); err != nil {
			fmt.Printf("Ошибка при сохранении HTML: %v\n", err)
		} else {
			fmt.Printf("HTML сохранен: %s\n", outputHTMLPath)
		}
		if err := saveToFile(outputCSSPath, extractedCSS); err != nil {
			fmt.Printf("Ошибка при сохранении CSS: %v\n", err)
		} else if extractedCSS != "" {
			fmt.Printf("CSS сохранен: %s\n", outputCSSPath)
		}
		if err := saveToFile(outputJSPath, extractedJS); err != nil {
			fmt.Printf("Ошибка при сохранении JS: %v\n", err)
		} else if extractedJS != "" {
			fmt.Printf("JS сохранен: %s\n", outputJSPath)
		}

	} else if *cleanCodeFlag {
		outputPath := *filePath
		if *outputDir != "" {
			outputPath = outputHTMLPath
			fmt.Printf("Сохранение очищенного файла в: %s\n", outputPath)
		} else {
			fmt.Printf("Перезапись очищенного файла: %s\n", outputPath)
		}

		if err := saveToFile(outputPath, processedHTML); err != nil {
			fmt.Printf("Ошибка при сохранении очищенного файла: %v\n", err)
		} else {
			fmt.Println("Очищенный файл успешно сохранен.")
		}
	}

	fmt.Println("Работа завершена.")
}

// cleanCode - функция для очистки HTML кода от нежелательного содержимого.
// Удаляет base64 URL, длинные base64 строки, длинные не-CSS стили внутри <style>,
// комментарии savepage, атрибуты data-savepage, скрипты data-savepage-type,
// пустые теги <style></style> и HTML комментарии.
// Возвращает очищенный HTML код.
func cleanCode(htmlContent string) string {
	cleanedCode := htmlContent

	// Удаление base64 URL в стилях.
	base64UrlRegex := regexp.MustCompile(`url\(\s*['"]?data:[^;]+;base64,[A-Za-z0-9+/=]+['"]?\s*\)`)
	cleanedCode = base64UrlRegex.ReplaceAllString(cleanedCode, "url('data:removed')")

	// Удаление длинных base64 строк вне URL (например, в атрибутах src).
	longBase64Regex := regexp.MustCompile(`data:[^;]+;base64,[A-Za-z0-9+/=]{200,}`)
	cleanedCode = longBase64Regex.ReplaceAllString(cleanedCode, "data:removed_long_base64")

	// Удаление длинных не-CSS контентов внутри тегов <style>, вероятно, мусора.
	longStyleRegex := regexp.MustCompile(`<style[^>]*>(?s:(.*?))</style>`)
	cleanedCode = longStyleRegex.ReplaceAllStringFunc(cleanedCode, func(styleTag string) string {
		styleContentRegex := regexp.MustCompile(`<style[^>]*>(?s:(.*?))</style>`)
		match := styleContentRegex.FindStringSubmatch(styleTag)
		if len(match) > 1 && len(match[1]) > 5000 && (!strings.Contains(match[1], "{") || !strings.Contains(match[1], "}")) {
			fmt.Println("Info: Removing potentially large non-CSS content within <style> tag.")
			return ""
		}
		return styleTag
	})

	// Удаление комментариев savepage.
	savePageCommentRegex := regexp.MustCompile(`(?s)/\*savepage-.*?-\*/`)
	cleanedCode = savePageCommentRegex.ReplaceAllString(cleanedCode, "")

	// Удаление атрибутов data-savepage-*.
	savePageAttrRegex := regexp.MustCompile(`\s*data-savepage-[^=]+="[^"]*"\s*`)
	cleanedCode = savePageAttrRegex.ReplaceAllString(cleanedCode, " ")

	// Удаление скриптов <script data-savepage-type="...">.
	savePageScriptRegex := regexp.MustCompile(`(?s)<script[^>]*data-savepage-type[^>]*>.*?</script>`)
	cleanedCode = savePageScriptRegex.ReplaceAllString(cleanedCode, "")

	// Удаление пустых тегов <style></style>.
	cleanedCode = strings.ReplaceAll(cleanedCode, "<style></style>", "")
	cleanedCode = regexp.MustCompile(`(?s)<style\s*>\s*</style>`).ReplaceAllString(cleanedCode, "")

	// Удаление HTML комментариев.
	htmlCommentRegex := regexp.MustCompile(`<!--.*?-->`)
	cleanedCode = htmlCommentRegex.ReplaceAllString(cleanedCode, "")

	// Удаление пустых строк в начале.
	cleanedCode = regexp.MustCompile(`^\s*\n`).ReplaceAllString(cleanedCode, "")

	return strings.TrimSpace(cleanedCode)
}

// separateContent - функция для разделения HTML контента на HTML, CSS и JS.
// Извлекает содержимое тегов <style> и <script> (без атрибута src) и возвращает
// HTML без этих тегов, а также извлеченный CSS и JS контент.
// Возвращает три строки: html, css, js.
func separateContent(htmlContent string) (html, css, js string) {
	html = htmlContent
	var cssBuilder strings.Builder
	var jsBuilder strings.Builder

	// Извлечение CSS из тегов <style>.
	styleRegex := regexp.MustCompile(`(?s)<style(?P<attrs>[^>]*)>(?P<content>.*?)</style>`)
	html = styleRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := styleRegex.FindStringSubmatch(match)
		if len(submatches) == 3 {
			content := strings.TrimSpace(submatches[2])
			if content != "" {
				cssBuilder.WriteString(content)
				cssBuilder.WriteString("\n\n")
				return "" // Удаляем тег <style> из HTML.
			}
		}
		return "" // Возвращаем пустую строку, если не удалось извлечь CSS.
	})

	// Извлечение JS из тегов <script> (без атрибута src).
	scriptRegex := regexp.MustCompile(`(?s)<script(?P<attrs>[^>]*)>(?P<content>.*?)</script>`)
	html = scriptRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := scriptRegex.FindStringSubmatch(match)
		if len(submatches) == 3 {
			attrs := submatches[1]
			content := strings.TrimSpace(submatches[2])
			// Извлекаем только inline скрипты (без src).
			if !regexp.MustCompile(`\ssrc\s*=`).MatchString(attrs) && content != "" {
				jsBuilder.WriteString(content)
				jsBuilder.WriteString("\n\n")
				return "" // Удаляем тег <script> из HTML.
			}
			if regexp.MustCompile(`\ssrc\s*=`).MatchString(attrs) {
				return match // Оставляем теги <script src="..."> как есть.
			}
		}
		return "" // Возвращаем пустую строку, если не удалось извлечь JS.
	})

	html = strings.TrimSpace(html)
	css = strings.TrimSpace(cssBuilder.String())
	js = strings.TrimSpace(jsBuilder.String())

	return html, css, js
}

// addLinksToHTML - функция для добавления ссылок на внешние CSS и JS файлы в HTML.
// Добавляет теги <link> для CSS и <script> для JS в <head> и <body> соответственно,
// если CSS и JS контент не пустой. Если теги <head> или <body> отсутствуют, они будут добавлены.
// Возвращает HTML с добавленными ссылками.
func addLinksToHTML(html, cssContent, jsContent, cssFileName, jsFileName string) string {

	// Добавление <!DOCTYPE html>, если отсутствует.
	if !regexp.MustCompile(`(?i)<\!doctype`).MatchString(html) {
		html = "<!DOCTYPE html>\n" + html
	}
	// Добавление <html></html>, если отсутствует.
	if !regexp.MustCompile(`(?i)<html`).MatchString(html) {
		html = "<html>\n" + html + "\n</html>"
	}
	hasHead := regexp.MustCompile(`(?i)</head>`).MatchString(html)
	// Добавление <head> с <meta charset="UTF-8">, если отсутствует.
	if !hasHead {
		if regexp.MustCompile(`(?i)<body`).MatchString(html) {
			html = regexp.MustCompile(`(?i)(<body.*?>)`).ReplaceAllString(html, "<head>\n<meta charset=\"UTF-8\">\n</head>\n$1")
		} else {
			html = regexp.MustCompile(`(?i)(<html[^>]*>)`).ReplaceAllString(html, "$1\n<head>\n<meta charset=\"UTF-8\">\n</head>")
		}
		hasHead = true
	} else {
		// Добавление <meta charset="UTF-8"> в <head>, если отсутствует.
		if !regexp.MustCompile(`(?i)<meta[^>]+charset`).MatchString(html) {
			html = regexp.MustCompile(`(?i)(<head[^>]*>)`).ReplaceAllString(html, "$1\n<meta charset=\"UTF-8\">")
		}
	}

	hasBody := regexp.MustCompile(`(?i)</body`).MatchString(html)
	// Добавление <body></body>, если отсутствует.
	if !hasBody {
		html = regexp.MustCompile(`(?i)(</html>)`).ReplaceAllString(html, "<body>\n</body>\n$1")
		hasBody = true
	}

	// Добавление ссылки на CSS файл в <head>, если CSS контент не пустой.
	if cssContent != "" {
		cssLink := `<link rel="stylesheet" href="` + cssFileName + `">`
		if hasHead {
			html = regexp.MustCompile(`(?i)</head>`).ReplaceAllString(html, cssLink+"\n</head>")
		} else {
			html = cssLink + "\n" + html
		}
	}

	// Добавление ссылки на JS файл в <body> в конце, если JS контент не пустой.
	if jsContent != "" {
		jsScript := `<script src="` + jsFileName + `"></script>`
		if hasBody {
			html = regexp.MustCompile(`(?i)</body>`).ReplaceAllString(html, jsScript+"\n</body>")
		} else {
			html = html + "\n" + jsScript
		}
	}

	return html
}

// saveToFile - функция для сохранения контента в файл.
// Пропускает сохранение пустых CSS и JS файлов.
// Возвращает ошибку, если не удалось сохранить файл.
func saveToFile(filename, content string) error {
	// Пропуск сохранения пустых CSS и JS файлов.
	if content == "" && (strings.HasSuffix(filename, ".css") || strings.HasSuffix(filename, ".js")) {
		fmt.Printf("Info: Пропуск сохранения пустого файла %s\n", filename)
		return nil
	}
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("не удалось сохранить %s: %w", filename, err)
	}
	return nil
}

// CSSRule - структура для представления CSS правила.
// Содержит селектор, контент (тело правила), полное правило и тип правила.
type CSSRule struct {
	selector string
	content  string
	full     string
	ruleType string
}

// removeUnusedCSS - функция для удаления неиспользуемых CSS стилей из CSS контента.
// Анализирует CSS, парсит правила и проверяет использование селекторов в HTML.
// Сохраняет только используемые правила, @import, @font-face, @keyframes и @media блоки.
// Возвращает CSS контент с удаленными неиспользуемыми правилами.
func removeUnusedCSS(css, html string) string {
	if css == "" || html == "" {
		return css // Возвращает CSS без изменений, если CSS или HTML пустые.
	}

	fmt.Println("Начинаем удаление неиспользуемых CSS стилей (Внимание: точность ограничена)...")

	rules := parseCSS(css) // Парсинг CSS в слайс CSSRule.
	if len(rules) == 0 {
		fmt.Println("Предупреждение: Не удалось разобрать CSS правила.")
		return css // Возвращает CSS без изменений, если не удалось распарсить правила.
	}

	totalRules := len(rules)
	keptRulesCount := 0
	var resultCSS strings.Builder

	// Сохранение @import, @font-face и @keyframes правил без проверки использования.
	for _, rule := range rules {
		if rule.ruleType == "import" || rule.ruleType == "font-face" || rule.ruleType == "keyframes" {
			resultCSS.WriteString(rule.full)
			resultCSS.WriteString("\n\n")
			keptRulesCount++
		}
	}

	// Проверка и сохранение используемых selector правил.
	for _, rule := range rules {
		if rule.ruleType == "selector" {
			if isSelectorUsed(rule.selector, html) {
				resultCSS.WriteString(rule.full)
				resultCSS.WriteString("\n\n")
				keptRulesCount++
			}
		}
	}

	// Сохранение @media блоков целиком без проверки использования селекторов внутри.
	for _, rule := range rules {
		if rule.ruleType == "media" {
			fmt.Printf("Info: Сохранение блока @media целиком: %s\n", strings.Split(rule.selector, "{")[0])
			resultCSS.WriteString(rule.full)
			resultCSS.WriteString("\n\n")
			keptRulesCount++
		}
	}

	fmt.Printf("Удаление CSS завершено. Сохранено примерно %d из %d правил.\n",
		keptRulesCount, totalRules)

	return strings.TrimSpace(resultCSS.String())
}

// parseCSS - функция для парсинга CSS контента в слайс CSSRule.
// Разбивает CSS на отдельные правила, определяет тип каждого правила (selector, import, font-face, keyframes, media)
// и извлекает селектор и контент правила.
// Возвращает слайс CSSRule.
func parseCSS(css string) []CSSRule {
	var rules []CSSRule
	// Удаление CSS комментариев перед парсингом.
	css = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(css, "")

	// Регулярное выражение для поиска CSS блоков (правил или директив).
	re := regexp.MustCompile(`(?s)(@[a-zA-Z-]+(?:\s+[^;{]+)?(?:;|\s*\{.*?\})|[^@{][^{}]*\{.*?\})`)

	matches := re.FindAllString(css, -1) // Находим все CSS блоки.

	for _, block := range matches {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		var rule CSSRule
		rule.full = block // Сохраняем полное правило.

		// Определение типа правила и извлечение селектора и контента.
		if strings.HasPrefix(block, "@import") {
			rule.ruleType = "import"
			rule.selector = strings.TrimSpace(strings.TrimSuffix(block, ";")) // Селектор для @import - вся строка.
		} else if strings.HasPrefix(block, "@font-face") {
			rule.ruleType = "font-face"
			rule.selector = "@font-face"
			braceIndex := strings.Index(block, "{")
			if braceIndex != -1 {
				rule.content = strings.TrimSpace(block[braceIndex+1 : strings.LastIndex(block, "}")]) // Контент @font-face.
			}
		} else if strings.HasPrefix(block, "@keyframes") || strings.HasPrefix(block, "@-webkit-keyframes") {
			rule.ruleType = "keyframes"
			braceIndex := strings.Index(block, "{")
			if braceIndex != -1 {
				rule.selector = strings.TrimSpace(block[:braceIndex]) // Селектор @keyframes.
				rule.content = strings.TrimSpace(block[braceIndex+1 : strings.LastIndex(block, "}")]) // Контент @keyframes.
			} else {
				rule.selector = block
			}
		} else if strings.HasPrefix(block, "@media") {
			rule.ruleType = "media"
			braceIndex := strings.Index(block, "{")
			if braceIndex != -1 {
				rule.selector = strings.TrimSpace(block[:braceIndex]) // Селектор @media.
				rule.content = strings.TrimSpace(block[braceIndex+1 : strings.LastIndex(block, "}")]) // Контент @media.
			} else {
				rule.selector = block
			}
		} else {
			rule.ruleType = "selector"
			braceIndex := strings.Index(block, "{")
			if braceIndex != -1 {
				rule.selector = strings.TrimSpace(block[:braceIndex]) // CSS селектор.
				rule.content = strings.TrimSpace(block[braceIndex+1 : strings.LastIndex(block, "}")]) // CSS контент.
			} else {
				continue // Пропускаем правила без тела.
			}
		}

		if rule.selector == "" && rule.content == "" && rule.ruleType != "import" {
			continue // Пропускаем пустые правила (кроме @import).
		}

		rules = append(rules, rule) // Добавляем распарсенное правило в слайс.
	}

	return rules
}

// isSelectorUsed - функция для проверки, используется ли CSS селектор в HTML контенте.
// Проверяет наличие селектора в HTML, игнорирует некоторые динамические и общие селекторы,
// и разбирает составные селекторы на простые для проверки.
// Возвращает true, если селектор используется в HTML, и false в противном случае.
func isSelectorUsed(selector, html string) bool {
	if selector == "" {
		return false // Пустой селектор не используется.
	}

	// Всегда считаем используемыми общие селекторы и псевдо-классы/элементы.
	if selector == "*" || selector == "html" || selector == "body" ||
		strings.Contains(selector, ":hover") || strings.Contains(selector, ":focus") ||
		strings.Contains(selector, ":active") || strings.Contains(selector, ":visited") ||
		strings.Contains(selector, ":root") || strings.Contains(selector, ":nth-child") ||
		strings.Contains(selector, "::before") || strings.Contains(selector, "::after") {
		return true
	}

	selectors := splitSelectors(selector) // Разделяем составной селектор на простые.

	for _, s := range selectors {
		cleanSel := removePseudoClassesAndElements(s) // Удаляем псевдо-классы и элементы.
		if checkSingleSelectorInHTML(cleanSel, html) { // Проверяем каждый простой селектор.
			return true
		}
	}

	// Считаем используемыми селекторы с комбинаторами (>+~ ).
	if strings.ContainsAny(selector, ">+~ ") {
		return true
	}

	return false // Селектор не найден в HTML.
}

// splitSelectors - функция для разделения CSS селектора на отдельные селекторы по запятой.
// Возвращает слайс строк с отдельными селекторами.
func splitSelectors(selector string) []string {
	return strings.Split(selector, ",")
}

// removePseudoClassesAndElements - функция для удаления псевдо-классов и псевдо-элементов из CSS селектора.
// Используется для упрощения селектора перед поиском в HTML.
// Возвращает очищенный селектор.
func removePseudoClassesAndElements(selector string) string {
	sel := regexp.MustCompile(`:[a-zA-Z-]+(\(.*?\))`).ReplaceAllString(selector, "") // Удаление псевдо-классов с параметрами.
	sel = regexp.MustCompile(`:[a-zA-Z-]+`).ReplaceAllString(sel, "")            // Удаление псевдо-классов.
	sel = regexp.MustCompile(`::[a-zA-Z-]+`).ReplaceAllString(sel, "")           // Удаление псевдо-элементов.
	return strings.TrimSpace(sel)
}

// checkSingleSelectorInHTML - функция для проверки наличия простого CSS селектора в HTML контенте.
// Проверяет селекторы по ID, классу, атрибуту и тегу.
// Возвращает true, если селектор найден в HTML, и false в противном случае.
func checkSingleSelectorInHTML(cleanSelector, html string) bool {
	cleanSelector = strings.TrimSpace(cleanSelector)
	if cleanSelector == "" || cleanSelector == "*" || cleanSelector == "body" || cleanSelector == "html" {
		return true // Общие селекторы всегда считаются используемыми.
	}

	// Проверка селектора по ID (#id).
	if strings.HasPrefix(cleanSelector, "#") {
		id := cleanSelector[1:]
		return strings.Contains(html, fmt.Sprintf("id=\"%s\"", id)) || strings.Contains(html, fmt.Sprintf("id='%s'", id))
	}

	// Проверка селектора по классу (.class).
	if strings.HasPrefix(cleanSelector, ".") {
		className := cleanSelector[1:]
		classRegex := regexp.MustCompile(fmt.Sprintf(`(?:^|\s|["'])%s(?:$|\s|["'#.:])`, regexp.QuoteMeta(className))) // Регулярное выражение для поиска класса.
		htmlClassAttrRegex := regexp.MustCompile(`(?i)class\s*=\s*["']([^"']*)["']`)                               // Регулярное выражение для поиска атрибута class.
		matches := htmlClassAttrRegex.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 && classRegex.MatchString(match[1]) { // Проверяем наличие класса в атрибуте class.
				return true
			}
		}
		return false // Класс не найден.
	}

	// Проверка селектора по атрибуту ([attr] или [attr=value]).
	if strings.HasPrefix(cleanSelector, "[") && strings.HasSuffix(cleanSelector, "]") {
		attrSelector := cleanSelector[1 : len(cleanSelector)-1]
		attrName := regexp.MustCompile(`^[a-zA-Z0-9_-]+`).FindString(attrSelector) // Извлекаем имя атрибута.
		if attrName != "" {
			return strings.Contains(html, " "+attrName+"=") || // Проверяем наличие атрибута.
				strings.Contains(html, " "+attrName+">") ||
				strings.Contains(html, " "+attrName+" ")
		}
		return false // Атрибут не найден.
	}

	// Проверка селектора по тегу (tag).
	if regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString(cleanSelector) {
		return strings.Contains(html, "<"+cleanSelector) // Проверяем наличие тега.
	}

	return true // В случае сложных селекторов, считаем, что используется (для безопасности).
}