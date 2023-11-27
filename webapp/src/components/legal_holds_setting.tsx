import React, {useEffect, useState} from 'react';
import Client from "@/client";
import {CreateLegalHold, LegalHold} from '@/types'
import {IntlProvider} from "react-intl";

import CreateLegalHoldForm from "@/components/create_legal_hold_form";
import LegalHoldTable from "@/components/legal_hold_table";


const LegalHoldsSetting = () => {
    let [legalHoldsFetched, setLegalHoldsFetched] = useState(false);
    let [legalHoldsFetching, setLegalHoldsFetching] = useState(false);
    let [legalHolds, setLegalHolds] = useState(Array<LegalHold>());
    let [showCreateModal, setShowCreateModal] = useState(false);

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

    const onCreateClicked = () => {

    };

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
        <IntlProvider locale="en-US">
            <div
                style={{
                    padding: "28px 32px",
                    border: "1px solid rgba(var(--sys-center-channel-color-rgb), 0.08)",
                    background: "var(--sys-center-channel-bg)",
                    borderRadius: "4px",
                    boxShadow: "0 2px 3px rgba(0, 0, 0, 0.08)",
                }}>
                <div
                    style={{
                        color: "#3f4350",
                        fontFamily: "Metropolis",
                        fontSize: "18px",
                        fontWeight: "700",
                        lineHeight: "24px",
                    }}>
                    Legal Holds
                </div>
                <hr/>

                <div style={{
                    display: "flex",
                    margin: "100px",
                    justifyContent: "center",
                }}>
                    <button
                        type='submit'
                        data-testid='create'
                        id='createLegalHold'
                        className='btn btn-primary'
                        onClick={() => setShowCreateModal(true)}
                    >Create new
                    </button>
                </div>

                <LegalHoldTable
                    legalHolds={legalHolds}
                    releaseLegalHold={releaseLegalHold}
                />
                <CreateLegalHoldForm
                    createLegalHold={createLegalHold}
                    visible={showCreateModal}
                    onExited={() => setShowCreateModal(false)}
                />
            </div>
        </IntlProvider>
    );
}

export default LegalHoldsSetting;
