import React, {Fragment, useContext} from 'react'
import {Typography, Grid, IconButton} from '@material-ui/core'
import {Cached} from '@material-ui/icons'
import PropTypes from 'prop-types'
import {STAT, Get} from '../repository'
import {UserContext} from '../context/UserState'

function Main(props) {
    const [stat, setStat] = React.useState(
        {
            version:"",
            oneConnectionTimeout:0,
            maxResponce:0,
            timeUP:"",
            nowConnected:0,
            maxConcurentConnection:0,
            allConnection:0,
            allIP:{}
    });
    const context = useContext(UserContext);
    console.log(context);
    return (
    <Fragment>
        <Grid container direction="row" alignItems="stretch" justify="center">
            <Grid item xs={12} container justify="flex-end">
                <IconButton onClick={()=>{
                       Get(STAT,{key:"key", value: context.state.key})
                       .then(resp => {
                           console.log("Receive server info ", resp);
                           setStat(resp)
                       })
                       .catch(err => console.log(err));
                }}>
                    <Cached style={{fontSize: 40}}/>
                </IconButton>
            </Grid>
            <Grid item xs={6} container>
                <div style={{display: "flex", flexDirection: "column"}}>
                    <Typography variant="h5">Server statistics:</Typography>
                    <ul>{Object.keys(stat).map(e=>{return (<li key={e}>{`${e}: ${stat[e]}`}</li>)})}</ul>
                </div>
            </Grid>
            <Grid item xs={6} container direction="column">
                <Typography variant="h6">Last item</Typography>
                <Typography variant="h6">Next item</Typography>
            </Grid>
        </Grid>
    </Fragment>
    );
}

Main.propTypes = {
    token: PropTypes.shape({
            key: PropTypes.string, 
            value: PropTypes.string
        })
}

export default Main