import React from 'react'
import {CircularProgress} from '@material-ui/core'
import './App.css'

export default function Loader(props) {
    let styles = {}
    if(props.hidden) styles.display="none" 
    return (<div id="loader" style={styles}>
        <CircularProgress/>
    </div>);
}