package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Объявление флагов
	headOnly := flag.Bool("I", false, "Получить только заголовки ответа (HEAD-запрос).")
	requestMethod := flag.String("X", "GET", "Указать кастомный HTTP-метод (GET, POST, PUT, DELETE).")
	data := flag.String("d", "", "Данные для отправки в теле запроса (автоматически переключает метод на POST).")
	headerCustom := flag.String("H", "", "Добавить свой заголовок в формате \"Name: Value\".")
	infiniteLoop := flag.Bool("L", false, "Запустить БЕСКОНЕЧНУЮ отправку запросов (цикл зашит внутри утилиты).")

	// Вынесение всей справки на флаги -h и --help с упоминанием нового имени инструмента
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "=====================================================================\n")
		fmt.Fprintf(os.Stderr, "  TPING1 — Кастомный аналог cURL на Go со встроенным бесконечным циклом\n")
		fmt.Fprintf(os.Stderr, "=====================================================================\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n  tping1 [флаги] <URL>\n\n")
		fmt.Fprintf(os.Stderr, "Доступные флаги:\n")
		
		flag.PrintDefaults() 

		fmt.Fprintf(os.Stderr, "\nОсобенности и примеры использования:\n")
		fmt.Fprintf(os.Stderr, "  1. Стандартный запрос заголовков:\n")
		fmt.Fprintf(os.Stderr, "     tping1 -I google.com\n\n")
		fmt.Fprintf(os.Stderr, "  2. БЕСКОНЕЧНЫЙ опрос заголовков (Режим зациклен внутри утилиты):\n")
		fmt.Fprintf(os.Stderr, "     tping1 -I -L https://example.com\n\n")
		fmt.Fprintf(os.Stderr, "  3. Бесконечная отправка POST-данных:\n")
		fmt.Fprintf(os.Stderr, "     tping1 -X POST -d \"login=admin\" -L http://testsite.local\n\n")
		fmt.Fprintf(os.Stderr, "Для остановки бесконечного режима нажмите Ctrl + C.\n")
		fmt.Fprintf(os.Stderr, "=====================================================================\n")
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Ошибка: Не указан целевой URL.")
		fmt.Println("Используйте флаг -h или --help для вывода подробной справки.")
		os.Exit(1)
	}
	targetURL := args

	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		executeRequest(client, targetURL, *headOnly, *requestMethod, *data, *headerCustom)

		if !*infiniteLoop {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func executeRequest(client *http.Client, url string, headOnly bool, method string, data string, customHeader string) {
	var reqMethod string
	var bodyReader io.Reader

	if headOnly {
		reqMethod = "HEAD"
	} else {
		reqMethod = strings.ToUpper(method)
	}

	if data != "" {
		bodyReader = strings.NewReader(data)
		if method == "GET" && !headOnly {
			reqMethod = "POST"
		}
	}

	req, err := http.NewRequest(reqMethod, url, bodyReader)
	if err != nil {
		fmt.Printf("[%s] Ошибка создания запроса: %v\n", time.Now().Format("15:04:05"), err)
		return
	}

	if customHeader != "" {
		parts := strings.SplitN(customHeader, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts), strings.TrimSpace(parts))
		}
	}

	req.Header.Set("User-Agent", "curl/tping1-custom-v1.2")

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[%s] Ошибка соединения: %v\n", time.Now().Format("15:04:05"), err)
		return
	}
	defer resp.Body.Close()

	duration := time.Since(startTime).Milliseconds()

	fmt.Printf("\n--- [%s] Запрос выполнен за %d ms ---\n", time.Now().Format("15:04:05"), duration)
	fmt.Printf("%s %s\n", resp.Proto, resp.Status)

	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	if !headOnly {
		fmt.Println("\n[Тело ответа]:")
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Ошибка чтения тела: %v\n", err)
			return
		}
		fmt.Println(string(bodyBytes))
	}
}
