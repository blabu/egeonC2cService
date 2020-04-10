import React from 'react'
import {Paper, Grid, TextField, IconButton, Button, Checkbox} from '@material-ui/core'
import {AddCircle as AddButton} from '@material-ui/icons'
import Loader from '../Loader'
import {PERM} from '../repository'
import {UserContext} from '../context/UserState'

export default function Keys(props) {
    let [allUrl, setAllUrl] = React.useState([
        {url: "/", isWrite: false},
    ]);
    const [errorsState, setErrorsState] = React.useState({
        userName: false,
        newKey: false,
        confirmKey: false,
    })

    const context = React.useContext(UserContext);

    const [loading, setLoading] = React.useState(false);

    function formSubmitHandler(event) {
        event.preventDefault();
        const inputData = event.target.elements;
        const name = inputData["userName"].value;
        const token = inputData["newKey"].value;
        const confirmToken = inputData["confirmKey"].value;
        const urls = []
        const localErrorsState = {...errorsState}
        if(name.length < 4) {
            localErrorsState.userName = true;
        } else {
            localErrorsState.userName = false;
        }
        if(token.length < 6) {
            console.log("new Key is too short");
            localErrorsState.newKey = true;
            
        } else {
            localErrorsState.newKey = false;
        }
        if(token !== confirmToken) {
            console.log("Keys is not equal");
            localErrorsState.confirmKey = true;
        } else {
            localErrorsState.confirmKey = false;
        }
        setErrorsState(localErrorsState);
        if(localErrorsState.confirmKey || localErrorsState.newKey || localErrorsState.userName) return;

        for(let i=0;;i++) {
            if(inputData["url"+i]) {
                urls.push({
                    url: inputData["url"+i].value,
                    isWrite: inputData["isWrite"+i].checked,
                })
            } else {
                break;
            }
        }
        setLoading(true);
        const req = {
            name,
            token,
            urls,
        }
        //Post(PERM,req,{key:"key",value:context.state.key})
        console.log("Form post query with param ", PERM, {key:"key",value:context.state.key})
        console.log("Request body ", req);
        setTimeout(()=>setLoading(false), 1000);
    }
    if(loading) {
        return <Loader/>
    }
    return (
        <div>
        <Paper elevation={2}>
            <Grid container>
            <Grid zeroMinWidth item container xs={10} direction="row" justify="center" spacing={2} alignContent="stretch">
                <form onSubmit={formSubmitHandler}>
                    <h1>Create new api key</h1>
                    <div>
                    <TextField
                        name="userName"
                        label="Name"
                        helperText={errorsState.userName? "User name is invalid":"Insert name here"}
                        variant="standard"
                        error={errorsState.userName}
                    />
                    </div>
                    <div>
                    <TextField
                        label="New key"
                        name="newKey"
                        helperText={errorsState.newKey? "New key is incorrect" :"Insert new key"}
                        type="text"
                        variant="standard"
                        error={errorsState.newKey}
                    />
                    <TextField
                        label="Confirm key"
                        name="confirmKey"
                        helperText={errorsState.confirmKey? "Confirm key is not equal" :"Insert key again"}
                        type="text"
                        variant="standard"
                        error={errorsState.confirmKey}
                    />
                    </div>
                    {allUrl.map((e,idx)=>{
                        console.log("Append new field", e);
                        return (<div key={idx}>
                            <TextField
                                label="url"
                                name={"url"+idx}
                                helperText="Insert new url here"
                                variant="standard"
                                defaultValue={e.url}
                            />
                            <Checkbox 
                                color="primary"
                                disabled={false}    
                                name={"isWrite"+idx}
                                checked={e.isWrite}
                                onChange={event=>{
                                    const temp = [...allUrl];
                                    temp[idx].isWrite = event.target.checked;
                                    setAllUrl(temp);
                                }}
                            />
                        </div>);
                    })}
                    <div>
                        <Button type="submit" variant="contained" color="primary">OK</Button>
                    </div>
                </form>
            </Grid>
            <Grid item xs={2} container alignContent="flex-end">
                <IconButton onClick={() => {
                    console.log("Add new url field");
                    setAllUrl([...allUrl,{url: "/", isWrite: false}])
                }}><AddButton fontSize="large"/></IconButton>
            </Grid>
            </Grid>
        </Paper>
        </div>
    )
}