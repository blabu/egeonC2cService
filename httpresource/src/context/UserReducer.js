
/*Action types*/
const UPDATE_KEY = "UPDATE_KEY"
const CONSOLE_STATE = "CONSOLE_STATE"

/*
Registered handlers
Объект харнит в себе список всех обработчиков
Каждый обработчик - это функция принимающая два параметра state и action
state  - это предыдущее состояние
action - объект содержащий обязательное поле type (string) (по нему определяется какой обработчик будет вызван)
         рекомендуемым вторым параметром выступает объект payload в котором хранится вся информация для изменения состояния 
*/
const handlers = {
    [CONSOLE_STATE]: (state,action)=>{console.log("State: ", state); console.log("Action: ", action); return state;},
    DEFAULT: state=>state, // Обработчик по умолчанию. Будет вызван если не найдено подх
}

/**
 * 
 * @param {Название обработчика из списка handlers} actionType 
 * @param {Функция которая будет вызвана для данного обработчика
 *         Эта функция принимает два параметра state и ation, возвращает новый state} handler 
 */
function AddHandler(actionType, handler) {
    handlers[actionType] = handler;
}

/**
 * Функция определяет какой обрабтчик вызвать и возвращает его
 * @param {Состояние контекста} state 
 * @param {Объект с обязательным параметром type} action 
 */
function UserReducer(state, action) {
    const handler = handlers[action.type] || handlers.DEFAULT
    return handler(state,action);
}

export {UserReducer, AddHandler, UPDATE_KEY}

