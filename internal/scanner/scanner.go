package scanner

// HandlerLocation описывает найденный HTTP-обработчик.
type HandlerLocation struct {
	FuncName   string // имя функции (для методов без ресивера)
	RecvType   string // тип ресивера (пусто, если обычная функция)
	File       string // относительный путь с '/'
	HasSwagger bool   // есть ли уже Swagger-документация над функцией
}

// HandlerFinder находит обработчики в проекте.
type HandlerFinder interface {
	Find(root string) ([]HandlerLocation, error)
}

// Opts – настройки сканера.
type Opts struct {
	AllHandlers bool // искать все функции с сигнатурой, не фильтруя по маршрутам
}
