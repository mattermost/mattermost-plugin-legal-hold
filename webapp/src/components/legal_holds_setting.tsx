import React, {useEffect, useState} from 'react';
import Client from "@/client";
import {CreateLegalHold, LegalHold} from '@/types'

import CreateLegalHoldForm from "@/components/create_legal_hold_form";
import LegalHoldTable from "@/components/legal_hold_table";


const LegalHoldsSetting = () => {
    let [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    let [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    let [legalHolds, setLegalHolds] = useState(Array<LegalHold>());

    const createLegalHold = async (data: CreateLegalHold) => {
        console.warn("TODO: Create the Legal Hold");
        try {
            const response = await Client.createLegalHold(data);
            setLegalHoldsFetched(false);
            return response;
        } catch (error) {
            console.log(error);
            throw error;
        }
    };

    const releaseLegalHold = async (id: string) => {
        try {
            const response = await Client.releaseLegalHold(id);
            setLegalHoldsFetched(false);
            return response;
        } catch (error) {
            console.log(error);
            throw error;
        }
    }

    useEffect(() => {
        const fetchLegalHolds = async () => {
            try {
                setLegalHoldsFetching(true);
                const data = await Client.getLegalHolds();
                setLegalHolds(data);
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
            } catch (error) {
                setLegalHoldsFetching(false);
                setLegalHoldsFetched(true);
                console.error(error);
            }
        }

        if (!legalHoldsFetched && !legalHoldsFetching) {
            fetchLegalHolds().catch(console.error);
        }
    });

    return (
        <div>
            <LegalHoldTable
                legalHolds={legalHolds}
                releaseLegalHold={releaseLegalHold}
            />
            <CreateLegalHoldForm
                createLegalHold={createLegalHold}
            />
        </div>
    );
}

export default LegalHoldsSetting;
