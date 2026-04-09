# Product Requirements Document: Comment Threading Model

**Version**: 1.0  
**Date**: 2026-04-05  
**Author**: Sarah (Product Owner)  
**Quality Score**: 90/100

---

## Executive Summary

Roasti поддерживает комментарии с возможностью отвечать на конкретный reply, но сейчас ответы на ответы молча теряются — они сохраняются в БД, но не возвращаются API. Это ломает ожидание пользователей: сообщение отправлено, но нигде не видно.

Решение — плоская модель тредов в стиле Instagram/Telegram: все replies под рутовым комментарием возвращаются плоским списком, отсортированным хронологически. Поле `parent_id` сохраняется в ответе, чтобы клиент мог показать контекст (@username). Изменение затрагивает только слой чтения — схема БД и API-контракт остаются без изменений.

---

## Problem Statement

**Текущая ситуация:** API возвращает только прямые дочерние комментарии рутового комментария (`WHERE parent_id IN (rootIDs)`). Если пользователь отвечает на reply, его сообщение сохраняется в БД, но не попадает в ответ эндпоинта `GET /posts/{id}/comments`. Визуально сообщение исчезает.

**Предлагаемое решение:** Заменить плоский запрос прямых потомков на рекурсивный обход дерева (SQLite recursive CTE). Возвращать всех потомков рутового комментария плоским списком с сохранением `parent_id`.

**Бизнес-эффект:** Корректное отображение диалогов. Пользователи могут вести связные разговоры внутри треда, отвечая на конкретные реплики.

---

## User Stories & Acceptance Criteria

### Story 1: Ответ на reply

**As a** пользователь  
**I want to** ответить на конкретный reply внутри треда  
**So that** мой ответ виден остальным участникам в контексте

**Acceptance Criteria:**
- [ ] `POST /posts/{id}/comments` с `parent_id`, указывающим на reply (не рутовый комментарий), создаёт комментарий без ошибок
- [ ] Созданный комментарий возвращается в `GET /posts/{id}/comments` в списке replies рутового комментария
- [ ] `parent_id` в ответе указывает на непосредственного родителя (не на рут)
- [ ] Порядок replies — хронологический (created_at ASC)

### Story 2: Отображение контекста @mention

**As a** клиент (мобильное/веб-приложение)  
**I want to** знать, на чьё сообщение именно отвечает пользователь  
**So that** можно показать "@username" перед текстом reply

**Acceptance Criteria:**
- [ ] Каждый reply содержит `parent_id`
- [ ] Клиент самостоятельно резолвит @username по `parent_id` из списка replies
- [ ] Если родительский reply удалён (`is_deleted: true`), логика отображения @mention остаётся на стороне клиента

### Story 3: Удалённый комментарий в дереве

**As a** пользователь  
**I want to** видеть цепочку разговора целиком  
**So that** контекст диалога не теряется при удалении одного из участников

**Acceptance Criteria:**
- [ ] Удалённый reply возвращается с `is_deleted: true`, пустым `text` и без `author`
- [ ] Replies на удалённый комментарий остаются видимыми и корректными
- [ ] Удалённый рутовый комментарий возвращается как tombstone; его replies сохраняются

---

## Functional Requirements

### Модель ответа

Структура ответа не меняется — `CommentThread` с плоским списком `replies`:

```
CommentThread {
  id, text, author, created_at, updated_at, is_deleted
  replies: [
    PostComment { id, text, author, parent_id, created_at, updated_at, is_deleted },
    PostComment { id, text, author, parent_id, created_at, updated_at, is_deleted },
    ...
  ]
}
```

`parent_id` в reply может указывать как на рутовый комментарий, так и на другой reply. Клиент использует это для рендеринга @mention.

### Поведение replies

| Сценарий | Поведение |
|---|---|
| Reply на рутовый комментарий | Включается в `replies` |
| Reply на reply | Включается в `replies` того же рутового комментария |
| Reply на удалённый reply | Включается в `replies`; родитель показывается как tombstone |
| Удалённый reply | Включается как tombstone (`is_deleted: true`) |

### Сортировка

Все replies отсортированы по `created_at ASC` — самые старые сверху.

### Пагинация replies

Все replies загружаются вместе с рутовыми комментариями в одном запросе. Отдельная пагинация replies не предусмотрена.

### Ограничения глубины

Глубина вложенности на уровне API не ограничена. Клиент не обязан отображать вложенность визуально — достаточно `parent_id` для контекста.

### Out of Scope

- Уведомления при ответе на reply (отдельная фича)
- Визуальная вложенность на стороне клиента (решение клиентских команд)
- Ограничение глубины ответов
- Отдельный эндпоинт пагинации replies

---

## Technical Constraints

### Изменения в слое чтения

Текущий запрос replies:
```sql
WHERE comments.parent_id IN (rootIDs)
```

Заменяется на рекурсивный обход:
```sql
WITH RECURSIVE descendants AS (
  SELECT id FROM comments WHERE parent_id IN (rootIDs)
  UNION ALL
  SELECT c.id FROM comments c
  JOIN descendants d ON c.parent_id = d.id
)
SELECT ... FROM comments WHERE id IN (SELECT id FROM descendants)
ORDER BY created_at ASC
```

### Совместимость

- Схема БД не меняется
- API-контракт не меняется (breaking change отсутствует)
- Изменения только в `internal/comments/repository.go`

### Производительность

SQLite поддерживает recursive CTE. При большом количестве replies на один тред нагрузка возрастает, но для MVP приемлемо. Если треды вырастут до сотен replies — добавить индекс по `parent_id` или ввести пагинацию replies.

---

## MVP Scope

**В MVP:**
- Recursive CTE в `ListForTarget` для получения всех потомков рутовых комментариев
- Плоский список replies с сохранением `parent_id`
- Хронологическая сортировка

**Пост-MVP:**
- Пагинация replies при больших тредах
- Уведомления при ответе на конкретный reply

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Циклические ссылки в parent_id | Low | High | Валидировать `parent_id` при создании — проверять `ExistsInTarget` уже есть |
| Производительность при глубоких тредах | Low | Medium | Индекс по `parent_id`; пагинация replies в пост-MVP |
| Клиенты не ожидают replies на replies | Low | Low | Breaking change отсутствует; `parent_id` уже в модели |

---

*PRD создан через интерактивный сбор требований с оценкой качества.*
