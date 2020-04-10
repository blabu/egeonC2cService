import React, { useState, useContext } from 'react'
import { Map as LeafletMap, Marker, Popup, TileLayer } from 'react-leaflet'
import Loader from '../Loader'
import Leaflet from 'leaflet'
import 'leaflet/dist/leaflet.css'
import '../../public/locBlue.png'
import '../../public/shadow.png'
import {UserContext} from "../context/UserState"
import { SERVER_ADDR } from '../repository';

export default function Map({position}) {
    const icon = new Leaflet.Icon({
        iconUrl: 'locBlue.png',
        iconSize:[25,41],
        iconAnchor: [7, 25],
        shadowUrl: "shadow.png",
        shadowSize: [40,80],
        shadowAnchor: [7, 60],
    })
    const [loading, setLoading] = useState(false);
    const context = useContext(UserContext);
    const [pos, setPosition] = useState({lat:position[0], lng:position[1], zoom:13});
    const mapURL = SERVER_ADDR+'/api/v1/maps/{z}/{x}/{y}?key={accessToken}';
    const markerPull = []
    for(let i=0; i<10; i++) {
        const coord = (Math.random() -1 )/100
        markerPull.push(<Marker key={i} position={[pos.lat+coord, pos.lng+coord]} icon={icon} draggable={true}></Marker>);
    }
    const loadedMap = (
        <div>
            <Loader hidden={loading}/>
            <LeafletMap whenReady={()=>setLoading(false)} center={[pos.lat, pos.lng]} zoom={pos.zoom} style={{width:"100%", height:"50rem"}}>
                    <TileLayer
                        url={mapURL}
                        accessToken={context.state.key}
                    />
                    {markerPull}
            </LeafletMap>
        </div>
    );
    return loadedMap;
}
