import React, {useContext} from 'react'
import PropTypes from 'prop-types'
import {UserContext} from '../context/UserState'
import {Grid, Typography} from '@material-ui/core'

function Main(props) {
    const context = useContext(UserContext);
    return (
        <Grid container direction="row" spacing={4}>
            <Grid item xs={2}>
                <Typography variant="h6">Hello, {context.state.name}</Typography>
            </Grid>
        </Grid>
    );
}

Main.propTypes = {
    token: PropTypes.shape({
            key: PropTypes.string, 
            value: PropTypes.string
        })
}

export default Main