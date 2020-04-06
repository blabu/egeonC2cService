import React, {Fragment} from 'react'
import {Typography} from '@material-ui/core'
import { UserContext } from '../context/UserState';

export default function User(props) {
    const context = React.useContext(UserContext);
    return (
        <Fragment>
            <Typography variant="h6">Hello your key is {context.state.key}</Typography>
        </Fragment>
    )
}