import React from 'react'
import {Paper, Grid, TextField, IconButton, Button} from '@material-ui/core'
import {AddCircle as AddButton, CheckBox} from '@material-ui/icons'

export default function Keys(props) {
    let [allUrl, setAllUrl] = React.useState([
        {url: "/", isWrite: false},
    ]);
    const [checked, setChecked] = React.useState(false);
    
    return (
        <div>
        <Paper elevation={2}>
            <Grid container>
            <Grid zeroMinWidth item container xs={10} direction="row" justify="center" spacing={2} alignContent="stretch">
                <form>
                    <h1>Create new api key</h1>
                    <div>
                    <TextField
                        label="Name"
                        helperText="Insert name here"
                        variant="standard"
                    />
                    </div>
                    <div>
                    <TextField
                        label="New key"
                        helperText="Insert new key"
                        type="text"
                        variant="standard"
                    />
                    <TextField
                        label="Confirm key"
                        helperText="Insert key again"
                        type="text"
                        variant="standard"
                    />
                    </div>
                    {allUrl.map((e,idx)=>{
                        console.log("Append new field", e);
                        return (<div key={idx}>
                            <TextField
                                label="url"
                                helperText="Insert new url here"
                                variant="standard"
                                defaultValue={e.url}
                            />
                            
                        </div>);
                    })}
                    <div>
                        <Button variant="contained" color="primary">OK</Button>
                    </div>
                </form>
            </Grid>
            <Grid item xs={2} container alignContent="flex-end">
                <IconButton onClick={() => {
                    console.log("Add new url field");
                    setAllUrl([...allUrl,{url: "", isWrite: false}])
                }}><AddButton fontSize="large"/></IconButton>
            </Grid>
            </Grid>
        </Paper>
        </div>
    )
}