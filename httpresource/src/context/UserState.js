import React from 'react'
import {UserReducer, AddHandler, UPDATE_KEY} from './UserReducer'

const UserContext = React.createContext({key:"",id:0});

export default function UserState({value, children}) {
    const [state, dispatch] = React.useReducer( UserReducer, value );
    
    AddHandler(UPDATE_KEY, (state, {payload})=>{
        console.log("Handlers call", UPDATE_KEY);
        console.log("State",state);
        console.log("Action", payload);
        return {...payload};
    });
    const updateState = (isLogin, key)=> {
        return dispatch({type: UPDATE_KEY, payload: {isLogin, key}});
    }
    return (
        <UserContext.Provider value={ {state, updateState} }>
            {children}
        </UserContext.Provider>
    )
} 

export {UserContext};