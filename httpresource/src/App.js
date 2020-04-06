import React, { Component } from 'react';
import PropTypes from 'prop-types'
import UserState, { UserContext } from './context/UserState'
import Menu from './Menu';
import Authorize from './Authorize'

function AppWrapper() {
  const context = React.useContext(UserContext)
  if(context.state.key && context.state.key.length > 5 && context.state.isLogin ) {
    return <Menu/>
  }
  return <Authorize/>
}

class App extends Component {
  constructor(props) {
    super(props);
    this.name = props.name;
    console.log(props);
  }

  componentDidMount() {
    console.log("Create app component");
  }

  render() {
    return (
    <UserState value={ {isLogin: false, key:this.name} }>
      <AppWrapper/>
    </UserState>);
  }
}

App.propTypes = {
  name: PropTypes.string.isRequired,
}

export default App;
