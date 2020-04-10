import React from 'react';
import {UserContext} from '../context/UserState';
import { Grid, Typography, IconButton, Paper } from '@material-ui/core';
import {Cached} from '@material-ui/icons';
import {Get, STAT} from '../repository';

class ServerStat extends React.Component {
    state = {
        version:"",
        oneConnectionTimeout:0,
        maxResponce:0,
        timeUP:"",
        nowConnected:0,
        maxConcurentConnection:0,
        allConnection:0,
        allIP:[],
    }

    updateServerStat() {
        Get(STAT, {key:"key", value: this.context.state.key})
                       .then(resp => {
                           this.setState(resp)
                       })
                       .catch(err => console.log(err));
    }

    componentDidMount() {
        this.updateServerStat()
    }

    render() {
        return (
            <Grid container direction="row" alignItems="stretch" justify="center">
                <Grid item xs={12} container justify="flex-end">
                <IconButton onClick={()=>{this.updateServerStat();}}><Cached style={{fontSize: 40}}/></IconButton>
                </Grid>
                <Grid item xs={6} container>
                    <Paper>
                    <div style={{display: "flex", flexDirection: "column"}}>
                        <Typography variant="h5">Server statistics:</Typography>
                        <ul> 
                            <li key={1}>
                                <p>Server version: {this.state.version}</p>
                            </li>
                            <li key={2}>
                                <p>Max timeout for one connection: {Math.floor(this.state.oneConnectionTimeout / 1000000000)} seconds</p>
                            </li>
                            <li key={3}>
                                <p>Maximum responce time: {this.state.maxResponce}</p>
                            </li>
                            <li key={4}>
                               <p>Server up since: { (new Date(this.state.timeUP)).toLocaleString('ru-Ru') }</p>
                            </li>
                            <li key={5}>
                                <p>Now connected: {this.state.nowConnected}</p>
                            </li>
                            <li key={6}>
                                <p>Max concurrect connection: {this.state.maxConcurentConnection}</p>
                            </li>
                            <li key={7}>
                                <p>Connection for all time: {this.state.allConnection}</p>
                            </li>
                            <li key={8}>
                                    All IP: <ul>{
                                        Object.keys(this.state.allIP).map((el,index) => (
                                            <li key={this.state.allIP[el].IP}>
                                                IP: {this.state.allIP[el].IP},
                                                Count: {this.state.allIP[el].Count},
                                                Time for last connection: {(new Date(this.state.allIP[el].LastTime)).toLocaleString('ru-Ru') } 
                                            </li>))
                                    }</ul>
                            </li>
                            <li>
                                Last update time: {(new Date()).toLocaleString('ru-Ru')}
                            </li>
                        </ul>
                    </div>
                    </Paper>
                </Grid>
            </Grid>
        );
    }
}

ServerStat.contextType = UserContext;

export default ServerStat;

/**
<ul>{this.state.allIP.map(el =>  
                                            <li>{el.IP}: {el.Count} times. Last seen: {el.LastTime} </li>
                                    )} </ul>
 */