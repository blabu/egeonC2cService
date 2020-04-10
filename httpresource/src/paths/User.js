import React from 'react'
import {Grid, Typography, Paper} from '@material-ui/core'
import { UserContext } from '../context/UserState';

export default function User(props) {
    const context = React.useContext(UserContext);
    return (
        <Grid container direction="row" spacing={4}>
            <Grid item xs={2}>
                <Paper>
                    <Typography variant="h6">Hello, {context.state.name}</Typography>
                    <Typography variant="h6">Your key is {context.state.key}</Typography>
                    
                </Paper>
            </Grid>
        </Grid>
    )
}