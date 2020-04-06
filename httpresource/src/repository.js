import axios from 'axios';


const STAT = "info"
const CLIENT = "client"
const ALL_CLIENTS = "clients"
const CHECK_KEY = "checkKey"

const formURL = (command, ...param) => {
    const url = "https://localhost:6060"
    const requestParam = param.map(e=>`${e.key}=${e.value}`).join("&");
    return `${url}/api/v1/${command}?${requestParam}`
}

function ResolveAfter(timeout, ...resolveParam) {
    return new Promise(resolve=>{
        setTimeout(()=>{
            resolve(...resolveParam);
        }, timeout);
    })
}

function GetFetch(command, ...param) {
    return new Promise((resolve, reject) => {
        fetch(formURL(command, ...param),
        {
            method: "GET",
            mode: "cors",
            cache: "no-cache",
            headers: {"Content-Type":"application/json"},
        })
        .then(answer => {
            if(!answer.ok) {
                throw (new Error(`Responce status ${answer.status}, ${answer.statusText}`));
            }
            return answer.json()
        })
        .then(result => resolve(result))
        .catch(err => reject(err))
        .finally(()=>console.log("Finaly method in get is worked..."))
    })
}

function PostFetch(command, postData, ...param) {
    return new Promise((resolve, reject) => {
        fetch(formURL(command, param),
        {
            method: "POST",
            mode: "cors",
            cache: "no-cache",
            headers: {"Content-Type":"application/json"},
            body: postData,
        })
        .then(answer => answer.json())
        .then(result => resolve(result))
        .catch(err => reject(err))
        .finally(()=>console.log("Finaly method in post is worked"))
    })
}

function GetAxios(command, ...param) {
    return new Promise((resolve, reject)=>{
        axios.get(formURL(command, ...param))
        .then(answer=>resolve(answer.data))
        .catch(er=>reject(er))
    })
}

function PostAxios(command, postData, ...param) {
    return new Promise((resolve, reject) =>{
        axios.post(formURL(command, ...param), postData)
        .then(answer=>resolve(answer.data))
        .catch(er=>reject(er))
    })
}

export {GetAxios as Get, PostAxios as Post, ResolveAfter, STAT, CLIENT, ALL_CLIENTS, CHECK_KEY};