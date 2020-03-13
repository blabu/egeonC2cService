import React from 'react'
import MenuItem from './MenuItem.js'

class Menu extends React.Component {
  render() {
    const menu = this.props.menu;
    let resMenu = menu.map((elem)=> {
      return <MenuItem key={elem.id} name={elem.Name} clickHandler={()=>{console.log(`click on ${this.name}`)}}/>
    })
    return (
      <div>
      <div className="header">
        {resMenu}
      </div>
      <div className="header">
        {resMenu}
      </div>
      <div className="header">
        {resMenu}
      </div>
      <div className="header">
        {resMenu}
      </div>
      </div>
    );
  }
}

export default Menu 