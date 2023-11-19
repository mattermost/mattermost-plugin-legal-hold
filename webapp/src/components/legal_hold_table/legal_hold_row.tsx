import {LegalHold} from "@/types";
import React from "react";


interface LegalHoldRowProps {
    legalHold: LegalHold;
}

const LegalHoldRow = (props:LegalHoldRowProps) => {
    const lh = props.legalHold;
    return (
        <div>{lh.name}</div>
    );
}

export default LegalHoldRow;
