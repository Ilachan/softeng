import React, { useState } from "react";
import { ENROLLED } from "../lib/constants";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import { Icons } from "../lib/icons";

const MySchedule = () => {
    const [showModal, setShowModal] = useState(false);
  return (
    <div className="max-w-2xl mx-auto space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <h2 className="text-2xl font-bold text-slate-800 mb-6">
        Your Upcoming Sessions
      </h2>
      {ENROLLED.map((course) => (
        <Card
          key={course.id}
          className="p-6 flex items-center justify-between group"
        >
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-2xl bg-indigo-50 flex items-center justify-center text-2xl text-indigo-600">
              <Icons.Calendar />
            </div>
            <div>
              <h3 className="text-lg font-bold text-slate-800">
                {course.title}
              </h3>
              <p className="text-slate-500 text-sm flex items-center gap-2 mt-1">
                <span className="bg-indigo-50 text-indigo-700 px-2 py-0.5 rounded text-xs font-semibold">
                  Confirmed
                </span>
                • {course.time}
              </p>
            </div>
          </div>
          <Button
            variant="danger"
            onClick={() => setShowModal(true)}
            className="opacity-0 group-hover:opacity-100 transition-opacity"
          >
            Drop
          </Button>
        </Card>
      ))}
       {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm animate-in fade-in duration-200">
          <Card className="max-w-sm w-full p-6 text-center animate-in zoom-in-95 duration-200">
            <div className="w-12 h-12 bg-red-100 text-red-600 rounded-full flex items-center justify-center mx-auto mb-4">
              <Icons.Alert />
            </div>
            <h3 className="text-lg font-bold text-slate-900">
              Cancel Registration?
            </h3>
            <p className="text-slate-500 text-sm mt-2 mb-6">
              Are you sure you want to drop this class? You will lose your spot
              to the next person on the waitlist.
            </p>
            <div className="grid grid-cols-2 gap-3">
              <Button variant="secondary" onClick={() => setShowModal(false)}>
                Keep Spot
              </Button>
              <Button variant="danger" onClick={() => setShowModal(false)}>
                Yes, Drop
              </Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
};

export default MySchedule;
