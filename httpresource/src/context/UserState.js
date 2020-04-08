import React from 'react'
import {UserReducer, AddHandler, UPDATE_KEY} from './UserReducer'

const UserContext = React.createContext({key:"", name:""});

export default function UserState({value, children}) {
    /*
        На базе переданного начального значения регистрируем свой Reducer
        Функцию, которая принимает state и action и возвращает обработчик 
    */
    const [state, dispatch] = React.useReducer( UserReducer, value );    
    
    /*
     Добавляем новый обработчик для события UPDATE_KEY
     */
    AddHandler(UPDATE_KEY, (state, {payload})=>{ // Из ation с помощью авторазпаковки достаем payload
        console.log("Handlers call", UPDATE_KEY); // Логируем действие
        console.log("State",state);               // Логируем состояние
        console.log("Action payload", payload);   // Логируем новое состояние
        return {...payload};            // Изменение состояния
    });

    // Функция которая вызывает обработчик UPDATE_KEY
    const updateState = (isLogin, key, name)=> {
        return dispatch({type: UPDATE_KEY, payload: {isLogin, key, name}});
    }

    return (
        /*
            Все дети оборачиваются и получают контектст пользователя,
            а в value передаем наш Reducer и функцию по его изменению
        */
        <UserContext.Provider value={ {state, updateState} }>
            {children}
        </UserContext.Provider>
    )
} 

export {UserContext};