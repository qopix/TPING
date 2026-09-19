package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	headOnly := flag.Bool("I", false, "Получить только заголовки ответа (HEAD-запрос).")
	requestMethod := flag.String("X", "GET", "Указать кастомный HTTP-метод (GET, POST, PUT, DELETE).")
	data := flag.String("d", "", "Данные для отправки в теле запроса (автоматически переключает метод на POST).")
	headerCustom := flag.String("H", "", "Добавить свой заголовок в формате \"Name: Value\".")
	infiniteLoop := flag.Bool("L", false, "Запустить БЕСКОНЕЧНУЮ отправку запросов (цикл зашит внутри утилиты).")
	
	// ИЗМЕНЕНО: Теперь принимаем просто строку-число (например, "100", "01", "1000")
	delayStr := flag.String("s", "1000", "Задержка между бесконечными запросами В МИЛЛИСЕКУНДАХ (например: 01, 10, 500).")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "=====================================================================\n")
		fmt.Fprintf(os.Stderr, "  TPING1 — Кастомный аналог cURL на Go со встроенным бесконечным циклом\n")
		fmt.Fprintf(os.Stderr, "=====================================================================\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n  tping1 [флаги] <URL>\n\n")
		fmt.Fprintf(os.Stderr, "Доступные флаги:\n")
		flag.PrintDefaults() 
		fmt.Fprintf(os.Stderr, "\nПримеры изменения скорости запросов (в миллисекундах):\n")
		fmt.Fprintf(os.Stderr, "  tping1 -I -L -s 01 google.com     (Максимальная скорость, задержка 1 мс)\n")
		fmt.Fprintf(os.Stderr, "  tping1 -I -L -s 500 google.com    (Каждые 500 миллисекунд)\n")
		fmt.Fprintf(os.Stderr, "\nДля остановки бесконечного режима нажмите Ctrl + C.\n")
		fmt.Fprintf(os.Stderr, "=====================================================================\n")
	}

	flag.Parse()

	// Конвертируем строку с числом в полноценное целое число (int)
	msCount, err := strconv.Atoi(*delayStr)
	if err != nil || msCount < 0 {
		fmt.Printf("Ошибка: Неверное значение флага -s (%s). Укажите целое число миллисекунд.\n", *delayStr)
		os.Exit(1)
	}
	// Создаем задержку на основе полученных миллисекунд
	delay := time.Duration(msCount) * time.Millisecond

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

	// Настройка HTTP-клиента с отключением удержания соединений (Keep-Alive),
	// чтобы при сверхбыстром флуде каждый запрос гарантированно открывал новое соединение.
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	for {
		executeRequest(client, targetURL, *headOnly, *requestMethod, *data, *headerCustom)

		if !*infiniteLoop {
			break
		}

		// Спим указанное количество миллисекунд
		if msCount > 0 {
			time.Sleep(delay)
		}
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
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	req.Header.Set("User-Agent", "curl/tping1-custom-v1.4")

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
