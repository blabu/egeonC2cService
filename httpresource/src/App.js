import React from 'react';
import './Animate.css';
import './App.css';
import Menu from './Menu.js'

function App() {
  let [val, setVal] = React.useState(
    [{
      id: 0,
      Name: "Главная",
      Value: "Главная",
    },
    {
      id: 1,
      Name: "Объекты",
      Value: "Объекты",
    },
    {
      id: 2,
      Name: "Настройки",
      Value: "Настройки",
    },
])
  return (
    <div className="App">
      <Menu menu={val}/>
    </div>
  );
}

export default App;
