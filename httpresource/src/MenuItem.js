import React from 'react'


class MenuItem extends React.Component {
    constructor(props) {
        super(props)
        console.log(props);
        this.name = props.name;
        // This binding is necessary to make `this` work in the callback
        //this.clickHandler = props.clickHandler.bind(this);
        this.clickHandler = ()=>{props.clickHandler();}
        console.log(this);
    }

    render() {
        return <div className="block black textWhite" onClick={this.clickHandler}>{this.name}</div>
    }
}


export default MenuItem