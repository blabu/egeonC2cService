import React from 'react'
import { Button, Switch, TextField, CircularProgress } from '@material-ui/core'
import "./Auth.css"
import { ResolveAfter, Get, CHECK_KEY } from './repository'
import {UserContext} from './context/UserState'

class Authorize extends React.Component {
    state = {
        isError: false,
        isLoad: false,
        isRemember: false
    }

    componentDidMount() {
        console.log("Current context is ", this.context);
        if (localStorage.getItem("isRemember")) {
            this.setKey(localStorage.getItem("key"), localStorage.getItem("name"));
        }
    }

    handleOnSubmit = this.handleOnSubmitProd;

    handleOnSubmitTest() {
        this.setState({...this.state, isLoad: true});
        ResolveAfter(1000, this.insertedKey)
        .then((data)=>{
            this.setState({isLoad: false, isError: false})
            this.setKey(data.key, data.name);
        })
    }

    handleOnSubmitProd() {
        this.setState({ ...this.state, isLoad: true });
        ResolveAfter(2000)
            .then(() => Get(CHECK_KEY,
                { key: "key", value: this.insertedKey },
                { key: "path", value: "/" }))
            .then(answer => {
                if (answer.error) {
                    console.log(answer.error)
                    throw answer.error;
                }
                this.setKey(answer.key, answer.name);
                if (this.state.isRemember) {
                    localStorage.setItem("name", answer.name);
                    localStorage.setItem("key", answer.key);
                    localStorage.setItem("isRemember", true);
                } else {
                    localStorage.setItem("name", "");
                    localStorage.setItem("key", "");
                    localStorage.setItem("isRemember", false);
                }
                this.setState({ ...this.state, isLoad: false });
            })
            .catch(err => {
                console.log(err);
                this.setState({ isError: true, isLoad: false });
            })
    }

    keyChanges(event) {
        this.insertedKey = event.target.value;
        if(this.state.isError) this.setState({ ...this.state, isError: false });
    }

    handleChange(event) {
        this.setState({...this.state, isRemember : event.target.checked});
    }

    setKey(newKey, name) {
        this.context.updateState(true, newKey, name);
    }

    render() {
        if (!this.state.isLoad) {
            return (
                <div className="authForm">
                <form onSubmit={(event) => { event.preventDefault(); this.handleOnSubmit(event) }}>
                    <div></div>
                    <div>
                    <TextField
                        autoFocus={true}
                        label={this.state.isError ? "Error" : "Key"}
                        variant="outlined"
                        error={this.state.isError}
                        helperText={this.state.isError ? "Incorrect key!":"Insert a valid key please!"}
                        onChange={this.keyChanges.bind(this)} />
                    <Switch
                        onChange={this.handleChange.bind(this)}
                        checked={this.state.isRemember}
                        name={"isRemember"}
                        color="primary" />
                    </div>
                    <div>
                    <Button
                        variant="contained"
                        color="primary"
                        onClick={this.handleOnSubmit.bind(this)}>OK
                    </Button>
                    </div>
                </form>
                </div>
            );
        }
        return <div className="authForm"><CircularProgress/></div>
    }
}

Authorize.contextType = UserContext

export default Authorize