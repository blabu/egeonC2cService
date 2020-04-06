
/*Action types*/
const UPDATE_KEY = "UPDATE_KEY"
const CONSOLE_STATE = "CONSOLE_STATE"

/*Registered handlers*/
const handlers = {
    [CONSOLE_STATE]: (state,action)=>{console.log("State: ", state); console.log("Action: ", action); return state;},
    DEFAULT: state=>state,
}

function AddHandler(actionType, handler) {
    handlers[actionType] = handler;
}

const UserReducer = (state, action)=>{
    const handler = handlers[action.type] || handlers.DEFAULT
    return handler(state,action);
}

export {UserReducer, AddHandler, UPDATE_KEY}

