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
            {legalHolds.map((legalHold, index) => {
                return <LegalHoldRow legalHold={legalHold} key={index}/>
            })}
        </div>
    );
}

export default LegalHoldTable;
