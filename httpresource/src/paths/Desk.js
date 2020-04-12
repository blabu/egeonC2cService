import React from 'react'
import '../App.css'


function MovableItem({x, y, children}) {
    const [state, setState] = React.useState({
        posX: x,
        posY: y, 
        prevX:0, 
        prevY:0, 
        isSelected: false
    });
    
    function moveElem(e) {
        e.preventDefault();
        const dx = e.clientX - state.prevX;
        const dy = e.clientY - state.prevY;
        setState({
            posX: state.posX + dx,
            posY: state.posY + dy,
            prevX: e.clientX,
            prevY: e.clientY,
            isSelected:state.isSelected
        })
    }
    const newCoord = {
        top:state.posY+"px", left:state.posX+"px"
    };
    if(state.isSelected) {
        newCoord.boxShadow = "#335617 2px 2px 5px";
    }
    return (
        <div
            className="movableItem"
            style={newCoord}
            onMouseDown={(e)=>{
               e.preventDefault();
                if (!state.isSelected) setState({
                    ...state, 
                    prevX: e.clientX, 
                    prevY: e.clientY, 
                    isSelected: true
                });
            }}
            onMouseUp={(e)=>{
                e.preventDefault();
                if(state.isSelected) setState({...state, isSelected: false});
            }}
            onMouseLeave={()=>{
                setState({...state, isSelected: false});
            }}
            onMouseMove={ state.isSelected ? moveElem:null }
            >
                {children}
        </div>
    );
}


export default class Desk extends React.Component {
    componentDidMount() {
        console.log("Component Desk is mount");
    }
    render() {
        return (
        <div style={{width: "100%", height: "100rem"}}>
            <MovableItem x={150} y={200}><p>Hello world</p></MovableItem>
            <MovableItem x={750} y={660}/>
            <MovableItem x={450} y={450}/>
        </div>);
    }
}