package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-unidecode"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateUsername создает варианты username из имени и фамилии с поддержкой транслитерации
func GenerateUsername(firstName, lastName string) []string {
	// Транслитерируем и очищаем имя и фамилию
	first := transliterateAndClean(firstName)
	last := transliterateAndClean(lastName)

	// Если после транслитерации получилось пусто, используем fallback
	if first == "" || last == "" {
		return generateFallbackUsernames(firstName, lastName)
	}

	variants := []string{}

	// Основные варианты
	// 1. firstlast (ivanpetrov)
	if len(first+last) <= 20 && len(first+last) >= 3 {
		variants = append(variants, first+last)
	}

	// 2. first_last (ivan_petrov)
	if len(first+"_"+last) <= 20 && len(first+"_"+last) >= 4 {
		variants = append(variants, first+"_"+last)
	}

	// 3. first.last (ivan.petrov)
	if len(first+"."+last) <= 20 && len(first+"."+last) >= 4 {
		variants = append(variants, first+"."+last)
	}

	// 4. flast (ipetrov)
	if len(first) > 0 && len(last) > 0 {
		variant := string(first[0]) + last
		if len(variant) >= 3 && len(variant) <= 20 {
			variants = append(variants, variant)
		}
	}

	// 5. firstl (ivanp)
	if len(first) > 0 && len(last) > 0 {
		variant := first + string(last[0])
		if len(variant) >= 3 && len(variant) <= 20 {
			variants = append(variants, variant)
		}
	}

	// 6. f_last (i_petrov)
	if len(first) > 0 {
		variant := string(first[0]) + "_" + last
		if len(variant) >= 4 && len(variant) <= 20 {
			variants = append(variants, variant)
		}
	}

	// 7. first_l (ivan_p)
	if len(last) > 0 {
		variant := first + "_" + string(last[0])
		if len(variant) >= 4 && len(variant) <= 20 {
			variants = append(variants, variant)
		}
	}

	// 8. Только first если подходящий
	if len(first) >= 4 && len(first) <= 15 {
		variants = append(variants, first)
	}

	// 9. Только last если подходящий
	if len(last) >= 4 && len(last) <= 15 {
		variants = append(variants, last)
	}

	// Если не получилось создать нормальные варианты, используем fallback
	if len(variants) == 0 {
		return generateFallbackUsernames(firstName, lastName)
	}

	// Убираем дубликаты и фильтруем
	return filterAndDeduplicate(variants)
}

// transliterateAndClean транслитерирует текст в ASCII и очищает от ненужных символов
func transliterateAndClean(text string) string {
	if text == "" {
		return ""
	}

	// Транслитерация в ASCII
	transliterated := unidecode.Unidecode(text)

	// Приводим к нижнему регистру
	transliterated = strings.ToLower(transliterated)

	// Убираем пробелы и заменяем их на подчеркивания если нужно
	transliterated = strings.TrimSpace(transliterated)
	transliterated = strings.ReplaceAll(transliterated, " ", "")

	// Очищаем от всех символов кроме букв и цифр
	cleaned := cleanLatinString(transliterated)

	// Убираем числа в начале (username не должен начинаться с цифры)
	cleaned = regexp.MustCompile(`^[0-9]+`).ReplaceAllString(cleaned, "")

	// Ограничиваем длину
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
	}

	return cleaned
}

// cleanLatinString удаляет все символы кроме латинских букв и цифр
func cleanLatinString(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}
	return strings.ToLower(result.String())
}

// generateFallbackUsernames создает username когда транслитерация не сработала
func generateFallbackUsernames(firstName, lastName string) []string {
	variants := []string{}

	// Пробуем более агрессивную очистку оригинального текста
	cleanFirst := cleanAnyString(firstName)
	cleanLast := cleanAnyString(lastName)

	if cleanFirst != "" && cleanLast != "" {
		if len(cleanFirst+cleanLast) >= 3 && len(cleanFirst+cleanLast) <= 20 {
			variants = append(variants, cleanFirst+cleanLast)
		}
	}

	// Если все еще пусто, генерируем случайные варианты
	if len(variants) == 0 {
		variants = append(variants, generateRandomUsernames()...)
	}

	return variants
}

// cleanAnyString пытается извлечь любые латинские символы из строки
func cleanAnyString(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// Если это латинские символы, добавляем как есть
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				result.WriteRune(unicode.ToLower(r))
			}
		}
	}
	return result.String()
}

// generateRandomUsernames создает случайные username когда ничего не помогло
func generateRandomUsernames() []string {
	prefixes := []string{"user", "guest", "member", "person"}
	variants := []string{}

	for _, prefix := range prefixes {
		// Добавляем случайные числа
		for i := 0; i < 3; i++ {
			num := rand.Intn(9999) + 1000
			variant := fmt.Sprintf("%s%d", prefix, num)
			variants = append(variants, variant)
		}
	}

	return variants
}

// GenerateUsernameWithSuffix добавляет суффикс к username для уникальности
func GenerateUsernameWithSuffix(baseUsername string, existingUsernames []string) string {
	// Очищаем базовый username на всякий случай
	baseUsername = transliterateAndClean(baseUsername)

	if baseUsername == "" {
		baseUsername = "user"
	}

	// Создаем map для быстрой проверки
	exists := make(map[string]bool)
	for _, u := range existingUsernames {
		exists[strings.ToLower(u)] = true
	}

	// Если базовый username свободен
	if !exists[strings.ToLower(baseUsername)] {
		return baseUsername
	}

	// Пробуем с числами от 1 до 999
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s%d", baseUsername, i)
		if !exists[strings.ToLower(candidate)] && len(candidate) <= 50 {
			return candidate
		}
	}

	// Пробуем с годом
	year := time.Now().Year()
	candidate := fmt.Sprintf("%s%d", baseUsername, year)
	if !exists[strings.ToLower(candidate)] && len(candidate) <= 50 {
		return candidate
	}

	// Пробуем с random suffix
	for i := 0; i < 10; i++ {
		suffix := rand.Intn(9999) + 1000
		candidate := fmt.Sprintf("%s%d", baseUsername, suffix)
		if !exists[strings.ToLower(candidate)] && len(candidate) <= 50 {
			return candidate
		}
	}

	// Последняя попытка с timestamp
	timestamp := time.Now().Unix() % 100000
	return fmt.Sprintf("%s%d", baseUsername, timestamp)
}

// GenerateAlternatives создает альтернативные варианты для занятого username
func GenerateAlternatives(username, firstName, lastName string, count int) []string {
	// Транслитерируем все входные данные
	username = transliterateAndClean(username)
	cleanFirst := transliterateAndClean(firstName)
	cleanLast := transliterateAndClean(lastName)

	alternatives := []string{}

	// Добавляем числовые суффиксы к оригинальному username
	for i := 1; i <= 5 && len(alternatives) < count; i++ {
		alt := fmt.Sprintf("%s%d", username, i)
		if len(alt) <= 50 {
			alternatives = append(alternatives, alt)
		}
	}

	// Добавляем год
	year := time.Now().Year()
	yearAlt := fmt.Sprintf("%s%d", username, year)
	if len(yearAlt) <= 50 && len(alternatives) < count {
		alternatives = append(alternatives, yearAlt)
	}

	// Добавляем короткий год
	shortYear := year % 100
	shortYearAlt := fmt.Sprintf("%s%02d", username, shortYear)
	if len(shortYearAlt) <= 50 && len(alternatives) < count {
		alternatives = append(alternatives, shortYearAlt)
	}

	// Пробуем другие комбинации имени и фамилии
	if cleanFirst != "" && cleanLast != "" {
		baseVariants := GenerateUsername(firstName, lastName)
		for _, variant := range baseVariants {
			if variant != username && len(alternatives) < count {
				alternatives = append(alternatives, variant)
			}
		}
	}

	// Добавляем рандомные числа
	for len(alternatives) < count {
		randNum := rand.Intn(999) + 100
		alt := fmt.Sprintf("%s%d", username, randNum)
		if len(alt) <= 50 {
			alternatives = append(alternatives, alt)
		} else {
			break
		}
	}

	return filterAndDeduplicate(alternatives[:min(count, len(alternatives))])
}

// filterAndDeduplicate убирает дубликаты и фильтрует некачественные варианты
func filterAndDeduplicate(variants []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, v := range variants {
		v = strings.ToLower(strings.TrimSpace(v))

		// Пропускаем если слишком короткий, слишком длинный, или уже есть
		if len(v) < 3 || len(v) > 50 || seen[v] {
			continue
		}

		// Пропускаем если состоит только из цифр
		if regexp.MustCompile(`^[0-9]+$`).MatchString(v) {
			continue
		}

		// Пропускаем если начинается с цифры
		if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
			continue
		}

		seen[v] = true
		result = append(result, v)
	}

	return result
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TransliterateText транслитерирует любой Unicode текст в ASCII
func TransliterateText(text string) string {
	if text == "" {
		return ""
	}
	return unidecode.Unidecode(text)
}
