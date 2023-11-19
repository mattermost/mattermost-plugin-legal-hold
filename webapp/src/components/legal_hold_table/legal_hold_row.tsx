import {LegalHold} from "@/types";
import React from "react";


interface LegalHoldRowProps {
    legalHold: LegalHold;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = (new Date(lh.ends_at)).toLocaleDateString();

    return (
        <React.Fragment>
            <div>{lh.name}</div>
            <div>{lh.display_name}</div>
            <div>{startsAt}</div>
            <div>{endsAt}</div>
            <div>{lh.user_ids}</div>
            <div><a href="#">Edit</a></div>
        </React.Fragment>
    );
}

export default LegalHoldRow;
