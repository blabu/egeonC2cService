import React from 'react'
import {AppBar, Toolbar, IconButton, Grid, Badge} from '@material-ui/core'
import {VpnKey, AccountCircle, ExitToApp, Home as HomeIcon, InfoOutlined, MapOutlined} from '@material-ui/icons'
import {BrowserRouter, Route, Switch, Link as RouterLink} from 'react-router-dom'
import Keys from './paths/Keys'
import User from './paths/User'
import Quit from './paths/Quit'
import Main from './paths/Main'
import Map  from './paths/Map'
import ServerInfo from './paths/ServerInfo'
import './App.css';
import {UserContext} from './context/UserState';

class Menu extends React.Component {
    componentDidMount() {
      console.log("Context in menu component", this.context);
    }

    render() {
        return (
            <BrowserRouter>
            <AppBar position="static" className="header">
              <Toolbar direction="row">
                <Grid container spacing={3}>
                  <Grid item sm={10}
                      container
                      direction="row"
                      justify="flex-start"
                      alignItems="baseline">
                    <IconButton color="inherit"
                      component={RouterLink} to="/">
                        <HomeIcon fontSize="large"/>
                    </IconButton>
                    <IconButton color="inherit"
                      component={RouterLink} to="/keys">
                        <Badge badgeContent={1} color="secondary"><VpnKey fontSize="large"/></Badge>
                    </IconButton>
                    <IconButton color="inherit"
                      component={RouterLink} to="/info">
                        <InfoOutlined fontSize="large"/>
                    </IconButton>
                    <IconButton color="inherit"
                      component={RouterLink} to="/map">
                        <MapOutlined fontSize="large"/>
                    </IconButton>
                  </Grid>
                  <Grid item sm={2}
                        container
                        direction="row"
                        justify="flex-end"
                        alignItems="baseline">
                    <IconButton color="inherit"
                      component={RouterLink} to="/user">
                      <AccountCircle fontSize="large"/>
                    </IconButton>
                    <IconButton color="inherit"
                      component={RouterLink} to="/exit">
                        <ExitToApp fontSize="large"/>
                    </IconButton>
                  </Grid>
                </Grid>
              </Toolbar>
            </AppBar>
            
            <Switch>
              <Route exact path="/">
                <Main/>
              </Route>
              <Route path="/keys">
                <Keys/>
              </Route>
              <Route path="/info">
                <ServerInfo/>
              </Route>
              <Route path="/map">
                <Map position={[50.448051, 30.521734]}/>
              </Route>
              <Route path="/user">
                <User/>
              </Route>
              <Route path="/exit">
                <Quit/>
              </Route>
            </Switch>
            </BrowserRouter>  
        )}
}

Menu.contextType = UserContext;

export default Menu