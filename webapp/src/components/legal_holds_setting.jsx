import React, {useEffect, useState} from 'react';
import Client from "client";

const LegalHoldsSetting = (props) => {
    let [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    let [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    let [legalHolds, setLegalHolds] = useState([]);

    useEffect(async () => {
        if (!legalHoldsFetched && !legalHoldsFetching) {
            try {
                setLegalHoldsFetching(true);
                const data = await Client.listLegalHolds();
                console.log(data);
                setLegalHolds(data);
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
            } catch (error) {
                //setLegalHoldsFetching(false);
                console.warn(error);
                setLegalHoldsFetched(true);
            }
        }
    });

    return (
        <div>
            <div>Hello World</div>
            {legalHolds.map((lh) => <div>{lh.id}</div>)}
        </div>);
}

export default LegalHoldsSetting;
