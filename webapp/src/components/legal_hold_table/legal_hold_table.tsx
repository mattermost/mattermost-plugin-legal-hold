import LegalHoldRow from '@/components/legal_hold_table/legal_hold_row';
import {LegalHold} from "@/types";
import React, {useEffect} from 'react';

interface LegalHoldTableProps {
    legalHolds: LegalHold[];
    actions: {
        getMissingProfilesByIds: Function,
    }
}


const LegalHoldTable = (props: LegalHoldTableProps) => {
    const legalHolds = props.legalHolds;

    const user_ids = Array.from(
        new Set(
            legalHolds
                .map((lh) => lh.user_ids)
                .reduce((prev, cur) => prev.concat(cur), [])
                .filter((i) => i !== null)
        )
    );
    console.log(user_ids);

    useEffect(() => {
        props.actions.getMissingProfilesByIds(
            user_ids
        ).then(console.log)
            .catch(console.error);
    });

    return (
        <div>
            <h3>Legal Holds</h3>
            <div
                style={{
                    display: "grid",
                    gridTemplateColumns: "auto auto auto auto auto auto"
                }}
            >
                <div style={{fontWeight: "bold"}}>Name</div>
                <div style={{fontWeight: "bold"}}>Slug</div>
                <div style={{fontWeight: "bold"}}>Start Date</div>
                <div style={{fontWeight: "bold"}}>End Date</div>
                <div style={{fontWeight: "bold"}}>Users</div>
                <div/>
                {legalHolds.map((legalHold, index) => {
                    return <LegalHoldRow legalHold={legalHold} key={index}/>
                })}
            </div>
        </div>
    );
}

export default LegalHoldTable;
