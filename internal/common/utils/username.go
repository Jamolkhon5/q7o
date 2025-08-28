package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateUsername создает варианты username из имени и фамилии
func GenerateUsername(firstName, lastName string) []string {
	// Очищаем и приводим к нижнему регистру
	first := cleanString(strings.ToLower(firstName))
	last := cleanString(strings.ToLower(lastName))

	if first == "" || last == "" {
		return []string{}
	}

	variants := []string{}

	// Основные варианты
	// 1. firstlast (ivanpetrov)
	if len(first+last) <= 20 {
		variants = append(variants, first+last)
	}

	// 2. first_last (ivan_petrov)
	if len(first+"_"+last) <= 20 {
		variants = append(variants, first+"_"+last)
	}

	// 3. first.last (ivan.petrov)
	if len(first+"."+last) <= 20 {
		variants = append(variants, first+"."+last)
	}

	// 4. flast (ipetrov)
	if len(first) > 0 && len(last) > 0 {
		variants = append(variants, string(first[0])+last)
	}

	// 5. firstl (ivanp)
	if len(first) > 0 && len(last) > 0 {
		variants = append(variants, first+string(last[0]))
	}

	// 6. f_last (i_petrov)
	if len(first) > 0 {
		variants = append(variants, string(first[0])+"_"+last)
	}

	// 7. first_l (ivan_p)
	if len(last) > 0 {
		variants = append(variants, first+"_"+string(last[0]))
	}

	// 8. Только first если короткий
	if len(first) >= 4 && len(first) <= 15 {
		variants = append(variants, first)
	}

	// 9. Только last если короткий
	if len(last) >= 4 && len(last) <= 15 {
		variants = append(variants, last)
	}

	// Убираем дубликаты
	return removeDuplicates(variants)
}

// GenerateUsernameWithSuffix добавляет суффикс к username для уникальности
func GenerateUsernameWithSuffix(baseUsername string, existingUsernames []string) string {
	// Создаем map для быстрой проверки
	exists := make(map[string]bool)
	for _, u := range existingUsernames {
		exists[u] = true
	}

	// Если базовый username свободен
	if !exists[baseUsername] {
		return baseUsername
	}

	// Пробуем с числами от 1 до 999
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s%d", baseUsername, i)
		if !exists[candidate] && len(candidate) <= 50 {
			return candidate
		}
	}

	// Пробуем с годом
	year := time.Now().Year()
	candidate := fmt.Sprintf("%s%d", baseUsername, year)
	if !exists[candidate] && len(candidate) <= 50 {
		return candidate
	}

	// Пробуем с random suffix
	for i := 0; i < 10; i++ {
		suffix := rand.Intn(9999)
		candidate := fmt.Sprintf("%s%d", baseUsername, suffix)
		if !exists[candidate] && len(candidate) <= 50 {
			return candidate
		}
	}

	// Последняя попытка с timestamp
	timestamp := time.Now().Unix() % 100000
	return fmt.Sprintf("%s%d", baseUsername, timestamp)
}

// GenerateAlternatives создает альтернативные варианты для занятого username
func GenerateAlternatives(username, firstName, lastName string, count int) []string {
	alternatives := []string{}

	// Добавляем числовые суффиксы
	for i := 1; i <= 3 && len(alternatives) < count; i++ {
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
	baseVariants := GenerateUsername(firstName, lastName)
	for _, variant := range baseVariants {
		if variant != username && len(alternatives) < count {
			alternatives = append(alternatives, variant)
		}
	}

	// Добавляем рандомные числа
	for len(alternatives) < count {
		randNum := rand.Intn(999) + 1
		alt := fmt.Sprintf("%s%d", username, randNum)
		if len(alt) <= 50 {
			alternatives = append(alternatives, alt)
		}
	}

	return removeDuplicates(alternatives)[:min(count, len(alternatives))]
}

// cleanString удаляет все не-буквенные символы
func cleanString(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// removeDuplicates удаляет дубликаты из слайса
func removeDuplicates(variants []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, v := range variants {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
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