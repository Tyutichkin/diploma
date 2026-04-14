#!/usr/bin/env python3
"""
Импорт задач из CSV или Excel в сервис планирования маршрутов.

Использование:
    python import_tasks.py --email you@example.com --password secret
    python import_tasks.py --email you@example.com --password secret --file my_tasks.csv
    python import_tasks.py --email you@example.com --password secret --file tasks.xlsx --clear

Формат файла (первая строка — заголовки, порядок столбцов произвольный):
    Столбец       Обязательный  Описание
    title         Да            Название задачи
    address       Нет           Адрес для геокодирования; если пусто — без карты
    duration      Нет           Целое число ≥ 1 (минуты); если пусто — без длительности
    window_start  Нет           Формат ЧЧ:ММ, например 09:00
    window_end    Нет           Формат ЧЧ:ММ, например 17:30

Поддерживаемые форматы: .csv (UTF-8, разделитель — запятая), .xlsx, .xls.
Excel-форматы требуют пакет openpyxl (pip install openpyxl).
"""

import argparse
import csv
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

DEFAULT_BASE_URL = "http://localhost:8080/api"
DEFAULT_FILE = Path(__file__).parent / "demo_tasks.csv"

# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------

def _request(method: str, url: str, body: dict | None = None, token: str | None = None) -> dict:
    data = json.dumps(body).encode() if body else None
    headers = {"Content-Type": "application/json", "Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            raw = resp.read()
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        body_text = e.read().decode(errors="replace")
        print(f"  HTTP {e.code} {e.reason}: {body_text}", file=sys.stderr)
        raise


# ---------------------------------------------------------------------------
# API calls
# ---------------------------------------------------------------------------

def login(base_url: str, email: str, password: str) -> str:
    print(f"Вход в систему как {email}...")
    try:
        resp = _request("POST", f"{base_url}/auth/login", {"email": email, "password": password})
    except urllib.error.HTTPError:
        print("Не удалось войти. Попробую зарегистрироваться...")
        resp = _request("POST", f"{base_url}/auth/register", {"email": email, "password": password})

    token = resp.get("accessToken") or resp.get("access_token")
    if not token:
        sys.exit("Авторизация не удалась. Проверьте email и пароль.")
    print("  OK")
    return token


def delete_all_tasks(base_url: str, token: str) -> None:
    print("Удаление существующих задач...")
    resp = _request("GET", f"{base_url}/tasks", token=token)
    tasks = resp if isinstance(resp, list) else resp.get("tasks", [])
    for t in tasks:
        _request("DELETE", f"{base_url}/tasks/{t['id']}", token=token)
    print(f"  Удалено задач: {len(tasks)}")


def create_task(base_url: str, token: str, row: dict, index: int) -> dict:
    title = row.get("title", "").strip()
    if not title:
        raise ValueError("Пустой title — строка пропущена")

    payload: dict = {
        "title":     title,
        "sortIndex": index,
    }

    # address — необязателен
    address = row.get("address", "").strip()
    if address:
        payload["addressText"] = address

    # duration — необязателен; если задан — целое число ≥ 1
    raw_dur = str(row.get("duration", "")).strip()
    if raw_dur:
        try:
            dur = int(raw_dur)
            if dur < 1:
                raise ValueError(f"duration должен быть ≥ 1, получено: {dur}")
            payload["durationMin"] = dur
        except ValueError as exc:
            raise ValueError(f"Некорректное значение duration {raw_dur!r}: {exc}") from exc

    # временны́е окна — необязательны
    ws = str(row.get("window_start", "")).strip()
    we = str(row.get("window_end",   "")).strip()
    if ws:
        payload["windowStartTime"] = ws
    if we:
        payload["windowEndTime"] = we

    return _request("POST", f"{base_url}/tasks", payload, token=token)


# ---------------------------------------------------------------------------
# File readers
# ---------------------------------------------------------------------------

def load_csv(path: Path) -> list[dict]:
    with open(path, newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        return [row for row in reader if any(v.strip() for v in row.values())]


def load_excel(path: Path) -> list[dict]:
    try:
        import openpyxl
    except ImportError:
        sys.exit(
            "Для чтения Excel нужен пакет openpyxl.\n"
            "Установите его: pip install openpyxl"
        )
    wb = openpyxl.load_workbook(path, read_only=True, data_only=True)
    ws = wb.active
    rows = list(ws.iter_rows(values_only=True))
    if not rows:
        return []
    headers = [str(h).strip() if h is not None else "" for h in rows[0]]
    result = []
    for row in rows[1:]:
        d = {headers[i]: (str(v).strip() if v is not None else "") for i, v in enumerate(row)}
        if any(d.values()):
            result.append(d)
    return result


def load_file(path: Path) -> list[dict]:
    suffix = path.suffix.lower()
    if suffix == ".csv":
        return load_csv(path)
    if suffix in (".xlsx", ".xls"):
        return load_excel(path)
    sys.exit(f"Неподдерживаемый формат файла: {suffix}. Используйте .csv, .xlsx или .xls")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(description="Импорт задач из CSV или Excel")
    parser.add_argument("--email",    required=True,  help="Email аккаунта")
    parser.add_argument("--password", required=True,  help="Пароль аккаунта")
    parser.add_argument("--file",     default=str(DEFAULT_FILE),
                        help="Путь к файлу .csv / .xlsx / .xls (default: demo_tasks.csv)")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="Базовый URL API")
    parser.add_argument("--clear",    action="store_true",
                        help="Удалить все существующие задачи перед импортом")
    args = parser.parse_args()

    file_path = Path(args.file)
    if not file_path.exists():
        sys.exit(f"Файл не найден: {file_path}")

    rows = load_file(file_path)
    if not rows:
        sys.exit("Файл пустой или не содержит данных.")

    print(f"Найдено строк: {len(rows)}")

    token = login(args.base_url, args.email, args.password)

    if args.clear:
        delete_all_tasks(args.base_url, token)

    print(f"\nИмпорт из {file_path.name}:")
    ok = 0
    for i, row in enumerate(rows, start=1):
        try:
            task = create_task(args.base_url, token, row, i)
            print(f"  [{i}/{len(rows)}] {task.get('title', row.get('title', '?'))!r}  → id={task.get('id', '?')}")
            ok += 1
        except Exception as exc:
            print(f"  [{i}/{len(rows)}] ОШИБКА: {row.get('title', '?')!r} — {exc}", file=sys.stderr)

    print(f"\nГотово: {ok}/{len(rows)} задач импортировано.")


if __name__ == "__main__":
    main()
