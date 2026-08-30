// Глобальный обработчик ошибок для очистки при перезагрузке

export function clearErrors() {
  // Очищаем ошибки из localStorage
  localStorage.removeItem('stickstock_errors');
  // Очищаем ошибки из sessionStorage
  sessionStorage.removeItem('stickstock_errors');
  // Очищаем ошибки из глобальной переменной
  if (typeof window !== 'undefined') {
    // @ts-ignore
    window.__stickstock_errors = [];
  }
}

export function getErrors(): any[] {
  if (typeof window === 'undefined') return [];
  // @ts-ignore
  return window.__stickstock_errors || [];
}

export function addError(error: any) {
  if (typeof window === 'undefined') return;
  // @ts-ignore
  if (!window.__stickstock_errors) {
    // @ts-ignore
    window.__stickstock_errors = [];
  }
  // @ts-ignore
  window.__stickstock_errors.push(error);
  // Сохраняем в localStorage для отладки
  try {
    // @ts-ignore
    localStorage.setItem('stickstock_errors', JSON.stringify(window.__stickstock_errors));
  } catch (e) {
    // ignore
  }
}

// Очищаем ошибки при загрузке страницы
if (typeof window !== 'undefined') {
  // Очищаем при каждой загрузке страницы
  clearErrors();
  
  // Очищаем при перезагрузке
  window.addEventListener('beforeunload', () => {
    clearErrors();
  });
}
