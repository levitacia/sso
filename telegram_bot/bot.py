import logging
import os
import json
import requests
from datetime import datetime
from dotenv import load_dotenv
from aiogram import Bot, Dispatcher, types, F
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.filters import Command
from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup
from aiogram.methods.send_message import SendMessage
import re

load_dotenv()

logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)

TOKEN = os.getenv("TELEGRAM_BOT_TOKEN")
SSO_SERVICE_URL = os.getenv("SSO_SERVICE_URL", "http://localhost:8080")

# переделать на БД, не хранить в питоновском словаре
user_tokens = {}

class AuthStates(StatesGroup):
    waiting_for_login = State()
    waiting_for_password = State()

bot = Bot(token=TOKEN)
dp = Dispatcher()

def escape_md(text):
    if text is None:
        return ""
    
    text = str(text)
    
    chars_to_escape = '_*[]()~`>#+-=|{}.!'
    for char in chars_to_escape:
        text = text.replace(char, f"\\{char}")
    
    return text

def format_logs_html(logs):
    if not logs:
        return "Логи не найдены."
    
    formatted_logs = "<b>📋 ЛОГИ АВТОРИЗАЦИИ</b>\n\n"
    
    for log in logs:
        timestamp = log.get("timestamp", "")
        if timestamp:
            try:
                dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                timestamp = dt.strftime("%Y-%m-%d %H:%M:%S")
            except:
                pass
        
        user = log.get("email", "Неизвестный пользователь")
        success = log.get("success", False)
        ip = log.get("ip", "")
        user_agent = log.get("user_agent", "")
        
        formatted_logs += f"🕒 <b>{timestamp}</b>\n"
        formatted_logs += f"👤 Пользователь: <code>{user}</code>\n"
        
        if success:
            formatted_logs += f"✅ Статус: <code>Успешно</code>\n"
        else:
            formatted_logs += f"❌ Статус: <code>Неуспешно</code>\n"
            
        formatted_logs += f"🌐 IP-адрес: <code>{ip}</code>\n"
        formatted_logs += f"🔹 Устройство входа: <code>{user_agent}</code>\n\n"
        formatted_logs += "-------------------\n\n"
    
    return formatted_logs

def get_main_keyboard():
    keyboard = [
        [InlineKeyboardButton(text="Войти", callback_data="login")],
        [InlineKeyboardButton(text="Получить логи", callback_data="get_logs")]
    ]
    return InlineKeyboardMarkup(inline_keyboard=keyboard)

@dp.message(Command("start"))
async def cmd_start(message: types.Message):
    await message.answer(
        "Бот загружает логи с сервиса авторизации.\n"
        "Выберите действие:",
        reply_markup=get_main_keyboard()
    )

@dp.callback_query(F.data == "login")
async def login_callback(callback: types.CallbackQuery, state: FSMContext):
    await callback.answer()
    await callback.message.edit_text("Введите логин:")
    await state.set_state(AuthStates.waiting_for_login)

@dp.message(AuthStates.waiting_for_login)
async def process_login(message: types.Message, state: FSMContext):
    await state.update_data(login=message.text)
    await message.answer("Пароль:")
    await state.set_state(AuthStates.waiting_for_password)

@dp.message(AuthStates.waiting_for_password)
async def process_password(message: types.Message, state: FSMContext):
    user_data = await state.get_data()
    login = user_data.get('login', '')
    password = message.text
    
    try:
        response = requests.post(
            f"{SSO_SERVICE_URL}/api/login",
            json={"email": login, "password": password}
        )
        
        if response.status_code == 200:
            token_data = response.json()
            token = token_data.get("token", "")
            user_tokens[message.from_user.id] = token
            
            await message.answer("Успех. Теперь можно получить логи")
        else:
            await message.answer(f"Ошибка авторизации: {response.text}")
    except Exception as e:
        await message.answer(f"Произошла ошибка при авторизации: {str(e)}")
    
    await message.answer("Выберите действие:", reply_markup=get_main_keyboard())
    await state.clear()

@dp.callback_query(F.data == "get_logs")
async def get_logs_callback(callback: types.CallbackQuery):
    await callback.answer()
    
    user_id = callback.from_user.id
    if user_id not in user_tokens:
        await callback.message.edit_text(
            "Вы не авторизованы. Используйте команду /start для начала."
        )
        return
    
    token = user_tokens[user_id]
    try:
        response = requests.get(
            f"{SSO_SERVICE_URL}/api/protected/logs",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        if response.status_code == 200:
            logs = response.json()
            
            formatted_logs = format_logs_html(logs)
            
            if len(formatted_logs) > 4000:
                for i in range(0, len(formatted_logs), 4000):
                    chunk = formatted_logs[i:i+4000]
                    try:
                        await callback.message.answer(
                            text=chunk,
                            parse_mode="HTML"
                        )
                    except Exception as e:
                        logger.error(f"Ошибка отправки HTML: {str(e)}")
                        await callback.message.answer(
                            text="Без форматирования"
                        )
                        await callback.message.answer(
                            text=re.sub(r'<[^>]+>', '', chunk)
                        )
            else:
                try:
                    await callback.message.answer(
                        text=formatted_logs,
                        parse_mode="HTML"
                    )
                except Exception as e:
                    logger.error(f"Ошибка отправки HTML: {str(e)}")
                    await callback.message.answer(
                        text="Без форматирования"
                    )
                    await callback.message.answer(
                        text=re.sub(r'<[^>]+>', '', formatted_logs)
                    )
        else:
            await callback.message.answer(f"Ошибка при получении логов: {response.text}")
    except Exception as e:
        await callback.message.answer(f"Произошла ошибка: {str(e)}")
    
    await callback.message.answer("Выберите действие:", reply_markup=get_main_keyboard())

@dp.message(Command("cancel"))
async def cmd_cancel(message: types.Message, state: FSMContext):
    current_state = await state.get_state()
    if current_state is None:
        return
    
    await state.clear()
    await message.answer("Операция отменена.")
    await message.answer("Выберите действие:", reply_markup=get_main_keyboard())

async def main():
    if not TOKEN:
        logger.error("Не указан токен бота в env (TELEGRAM_BOT_TOKEN)")
        return

    await dp.start_polling(bot)

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())