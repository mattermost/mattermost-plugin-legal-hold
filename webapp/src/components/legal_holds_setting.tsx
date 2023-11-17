import React, {useEffect, useState} from 'react';
import Client from "@/client";
import {LegalHold} from '@/types'
import CreateLegalHoldForm from "@/components/create_legal_hold_form";

const LegalHoldsSetting = () => {
    let [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    let [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    let [legalHolds, setLegalHolds] = useState(Array<LegalHold>());

    useEffect(() => {
        const fetchLegalHolds = async () => {
            try {
                setLegalHoldsFetching(true);
                const data = await Client.listLegalHolds();
                console.log(data);
                setLegalHolds(data);
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
            } catch (error) {
                setLegalHoldsFetching(false);
                console.warn(error);
                setLegalHoldsFetched(true);
            }
        }

        if (!legalHoldsFetched && !legalHoldsFetching) {
            fetchLegalHolds().catch(console.error);
        }
    });

    return (
        <div>
            <div>Hello World</div>
            {legalHolds.map((lh) => <div>{lh.name}</div>)}
            <CreateLegalHoldForm/>
        </div>);
}

export default LegalHoldsSetting;
